/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dal

import (
	"context"
)

import (
	"github.com/arana-db/arana/pkg/proto"
	"github.com/arana-db/arana/pkg/runtime/ast"
	rcontext "github.com/arana-db/arana/pkg/runtime/context"
	"github.com/arana-db/arana/pkg/runtime/namespace"
	"github.com/arana-db/arana/pkg/runtime/optimize"
	"github.com/arana-db/arana/pkg/runtime/plan/dal"
	"github.com/arana-db/arana/pkg/runtime/plan/dml"
	"github.com/arana-db/arana/pkg/security"
)

func init() {
	optimize.Register(ast.SQLTypeShowOpenTables, optimizeShowOpenTables)
}

func optimizeShowOpenTables(ctx context.Context, o *optimize.Optimizer) (proto.Plan, error) {
	var invertedIndex map[string]string
	for logicalTable, v := range o.Rule.VTables() {
		t := v.Topology()
		t.Each(func(x, y int) bool {
			if _, phyTable, ok := t.Render(x, y); ok {
				if invertedIndex == nil {
					invertedIndex = make(map[string]string)
				}
				invertedIndex[phyTable] = logicalTable
			}
			return true
		})
	}

	stmt := o.Stmt.(*ast.ShowOpenTables)

	clusters := security.DefaultTenantManager().GetClusters(rcontext.Tenant(ctx))
	plans := make([]proto.Plan, 0, len(clusters))
	for _, cluster := range clusters {
		ns := namespace.Load(cluster)
		// 配置里原子库 都需要执行一次
		groups := ns.DBGroups()
		for i := 0; i < len(groups); i++ {
			ret := dal.NewShowOpenTablesPlan(stmt)
			ret.BindArgs(o.Args)
			ret.SetInvertedShards(invertedIndex)
			ret.SetDatabase(groups[i])
			plans = append(plans, ret)
		}
	}

	return &dml.CompositePlan{
		Plans: plans,
	}, nil
}
