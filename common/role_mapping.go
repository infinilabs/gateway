// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package common

func GetLDAPGroupsMappingRoles(str []string) []string {
	var roles []string
	roles = append(roles, "admin")
	return roles
}

type Roles struct {
	ClusterAllowedRoles map[string][]Role
	ClusterDeniedRoles  map[string][]Role
}

type Role struct {
	Cluster []ClusterPermission //A list of cluster privileges.
	Indices []IndexPermission   //A list of indices permissions entries.
}

type ClusterPermission struct {
	Name string
	Path []string
}

type IndexPermission struct {
	Name                   []string
	Privileges             []string
	FieldSecurity          []string
	Query                  string
	AllowRestrictedIndices string
}

type FieldPermission struct {
	Name []string
	Type string //grant,deny,mask
}
