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

