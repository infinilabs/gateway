/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package native

import (
	"bytes"
	"fmt"
	"github.com/buger/jsonparser"
	"infini.sh/cloud/core/security"
	"infini.sh/cloud/modules/system/security/realm"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/elastic/adapter"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/dgraph-io/ristretto"

	"infini.sh/cloud/modules/system/security/realm/authc/native"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/lib/fasthttp"
)

type SecurityFilter struct {
	Elasticsearch string `config:"elasticsearch"`
	clusterUUID string
}

func (filter *SecurityFilter) Name() string {
	return "security"
}

const AccessDeniedMessage = "Access denied"
const NoUserMessage = "No user found"
const NoRoleMessage = "No roles found"

var rbacOnce sync.Once
var users *ristretto.Cache

func (filter *SecurityFilter) Filter(ctx *fasthttp.RequestCtx) {
	if filter.clusterUUID == "" {
		ctx.Error("can not find cluster uuid", 500)
		ctx.Finished()
		return
	}
	exists, username, password := ctx.Request.ParseBasicAuth()
	if !exists {
		ctx.Error(NoUserMessage, 401)
		ctx.Finished()
		return
	}
	user, err := getUser(string(username), string(password))
	if err != nil {
		log.Debugf("validate user error: %v", err)
		ctx.Error(AccessDeniedMessage, 401)
		ctx.Finished()
		return
	}
	var currentRoles []string
	for _, rl := range user.Roles {
		currentRoles = append(currentRoles, rl.Name)
	}
	if len(currentRoles) == 0 {
		ctx.Error(NoRoleMessage, 401)
		ctx.Finished()
		return
	}

	req := ctx.Request
	permission, params, matched := security.SearchAPIPermission("elasticsearch", string(req.Header.Method()), string(req.PhantomURI().Path()))
	if matched && permission != "" {
		tryRefreshRolesAndPermission()
		rolePermission := getRolePermissions(currentRoles)
		log.Debugf("currentRoles: %v, rolePermission :%v", currentRoles, rolePermission)
		if indexName, ok := params["index_name"]; ok {

			indices := strings.Split(indexName, ",")
			for _, index := range indices {
				indexReq := security.IndexRequest{
					Cluster:   filter.clusterUUID,
					Index:     index,
					Privilege: []string{permission},
				}

				err := security.ValidateIndex(indexReq, rolePermission)
				if err != nil {
					log.Debugf("validate index [%s] privilege error: %v", index, err)
					ctx.Error(AccessDeniedMessage, 401)
					ctx.Finished()
					return
				}
			}
		} else {
			clusterReq := security.ClusterRequest{
				Cluster:   filter.clusterUUID,
				Privilege: []string{permission},
			}
			if permission == "cluster.search" {
				hasAll, indices :=  security.GetRoleIndex(currentRoles, filter.clusterUUID)
				if !hasAll && len(indices) == 0 {
					log.Debugf("empty index privilege")
					ctx.Error(AccessDeniedMessage, 401)
					ctx.Finished()
					return
				}
				if !hasAll{
					body := ctx.Request.Body()
					if len(body) == 0 {
						body = []byte("{}")
					}
					v, _, _, _ := jsonparser.Get(body, "query")
					newQ := bytes.NewBuffer([]byte(`{"bool": {"must": [{"terms": {"_index":`))
					indicesBytes := util.MustToJSONBytes(indices)
					newQ.Write(indicesBytes)
					newQ.Write([]byte("}}"))
					if len(v) > 0 {
						newQ.Write([]byte(","))
						newQ.Write(v)
					}
					newQ.Write([]byte(`]}}`))
					body, _ = jsonparser.Set(body, newQ.Bytes(), "query")
					ctx.Request.SetBody(body)
				}
			}
			err := security.ValidateCluster(clusterReq, rolePermission)
			if err != nil {
				log.Debugf("validate cluster privilege error: %v", err)
				ctx.Error(AccessDeniedMessage, 401)
				ctx.Finished()
				return
			}
		}
	}
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("security", NewSecurityFilter, &SecurityFilter{})
}

func NewSecurityFilter(c *config.Config) (pipeline.Filter, error) {

	runner := SecurityFilter{}
	var err error

	if err = c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}
	runner.clusterUUID, err = adapter.GetClusterUUID(runner.Elasticsearch)
	if err != nil {
		return nil, fmt.Errorf("failted to init cluster uuid for cluster [%s]: %v", runner.Elasticsearch, err)
	}
	rbacOnce.Do(func() {
		users, err = ristretto.NewCache(&ristretto.Config{
			NumCounters: 1e5,     // Num keys to track frequency of (10M). 10,0000
			MaxCost:     1000000, //cfg.MaxCachedSize, // Maximum cost of cache (1GB).
			BufferItems: 64,      // Number of keys per Get buffer.
			Metrics:     false,
		})
	})

	return &runner, err
}

func getUser(username string, password string) (*security.User, error) {
	if v, ok := users.Get(username); ok {
		if su, ok := v.(*security.User); ok {
			return su, nil
		}
	}
	ok, user, err := realm.Authenticate(username, password, "")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("invalid username or password")
	}
	_, _ = realm.Authorize(user)
	users.SetWithTTL(user.Username, user, 0, time.Second * 30)
	return user, nil
}

var (
	roleMutex = sync.Mutex{}
	lastRefreshTime time.Time
)
func tryRefreshRolesAndPermission(){
	if !lastRefreshTime.IsZero() && lastRefreshTime.After(time.Now().Add(-time.Second*30)){
		return
	}
	roleMutex.Lock()
	defer roleMutex.Unlock()
	if !lastRefreshTime.IsZero() && lastRefreshTime.After(time.Now().Add(-time.Second*30)){
		return
	}
	native.Init()
	lastRefreshTime = time.Now()
}

func getRolePermissions(roleNames []string) security.RolePermission {
	rolePermission := security.CombineUserRoles(roleNames)
	//copy privilege used for map cluster_uuid privilege
	var newClusterPrivilege = security.ElasticsearchAPIPrivilege{}
	for clusterID, privileges := range rolePermission.ElasticPrivilege.Cluster {
		newClusterPrivilege[clusterID] = privileges
		clusterUUID, err := adapter.GetClusterUUID(clusterID)
		if err != nil {
			log.Debugf("get cluster uuid error: %v", err)
			continue
		}
		newClusterPrivilege[clusterUUID] = privileges
	}
	rolePermission.ElasticPrivilege.Cluster = newClusterPrivilege
	newIndexPrivilege := map[string]security.ElasticsearchAPIPrivilege{}
	for clusterID, privileges := range rolePermission.ElasticPrivilege.Index {
		newIndexPrivilege[clusterID] = privileges
		clusterUUID, err := adapter.GetClusterUUID(clusterID)
		if err != nil {
			log.Debugf("get cluster uuid error: %v", err)
			continue
		}
		newIndexPrivilege[clusterUUID] = privileges
	}
	rolePermission.ElasticPrivilege.Index = newIndexPrivilege
	return rolePermission
}