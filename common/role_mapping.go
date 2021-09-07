/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package common


func GetLDAPGroupsMappingRoles(str []string)[]string  {
	var roles []string
	roles=append(roles,"admin")
	return roles
}


type Roles struct {
	ClusterAllowedRoles map[string][]Role
	ClusterDeniedRoles  map[string][]Role
}

type Role struct {
	Cluster []ClusterPermission //A list of cluster privileges.
	Indices []IndexPermission //A list of indices permissions entries.
}

type ClusterPermission struct {
	Name string
	Path []string
}

type IndexPermission struct {
	Name []string
	Privileges []string
	FieldSecurity []string
	Query string
	AllowRestrictedIndices string
}

type FieldPermission struct {
	Name []string
	Type string //grant,deny,mask
}
