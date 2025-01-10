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
	"fmt"
	"testing"
	"time"
)

func TestUnixstamp(t *testing.T) {
	t1 := time.Now()
	fmt.Println(t1.Unix())
	fmt.Println(t1.Add(time.Second * 30).Unix())

	fmt.Println(time.Time{}.Unix())
}

func TestParseSegmentID(t *testing.T) {
	fileName := "_3g_Lucene85FieldsIndexfile_pointers_6x"
	fileName1 := "_3e.fdt"

	//3g>3e

	segmentID1 := ParseSegmentID(fileName)
	fmt.Println(segmentID1)

	segmentID2 := ParseSegmentID(fileName1)
	fmt.Println(segmentID2)

	fmt.Println(CompareSegmentIDs(segmentID1, segmentID2))
	fmt.Println(CompareSegmentIDs(segmentID2, segmentID1))
	fmt.Println(CompareSegmentIDs(segmentID2, segmentID2))
	fmt.Println(CompareSegmentIDs("12", "123"))

}
