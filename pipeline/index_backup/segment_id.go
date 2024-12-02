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

/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package index_backup

import (
	"infini.sh/framework/core/util"
	"strings"
)

func ParseSegmentID(fileName string)string  {
	if util.PrefixStr(fileName,"_"){
		arr:=strings.Split(fileName,"_")
		if len(arr)>1{
			firstPart:=arr[1]
			if util.ContainStr(firstPart,"."){
				arr:=strings.Split(firstPart,".")
				if len(arr)>0{
					segmentID:=arr[0]
					return segmentID
				}
			}
			return firstPart
		}
	}
	return ""
}

//The result will be:
// 0  if a==b,
//-1 if a < b,
//+1 if a > b.
func CompareSegmentIDs(id1,id2 string)int  {
	if len(id1)!=len(id2){
		if len(id1)>len(id2){
			return 1
		}else{
			return -1
		}
	}
	return  strings.Compare(id1,id2)
}

