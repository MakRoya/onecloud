// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SNatgatewayResourceBase struct {
	NatgatewayId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

type SNatgatewayResourceBaseManager struct {
	SVpcResourceBaseManager
}

func ValidateNatGatewayResourceInput(userCred mcclient.TokenCredential, input api.NatGatewayResourceInput) (*SNatGateway, api.NatGatewayResourceInput, error) {
	natObj, err := NatGatewayManager.FetchByIdOrName(userCred, input.Natgateway)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", NatGatewayManager.Keyword(), input.Natgateway)
		} else {
			return nil, input, errors.Wrap(err, "NatGatewayManager.FetchByIdOrName")
		}
	}
	input.Natgateway = natObj.GetId()
	return natObj.(*SNatGateway), input, nil
}

func (self *SNatgatewayResourceBase) GetNatgateway() (*SNatGateway, error) {
	obj, err := NatGatewayManager.FetchById(self.NatgatewayId)
	if err != nil {
		return nil, errors.Wrap(err, "NatGatewayManager.FetchById")
	}
	return obj.(*SNatGateway), nil
}

func (self *SNatgatewayResourceBase) GetVpc() *SVpc {
	nat, _ := self.GetNatgateway()
	if nat != nil {
		return nat.GetVpc()
	}
	return nil
}

func (self *SNatgatewayResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) api.NatGatewayResourceInfo {
	return api.NatGatewayResourceInfo{}
}

func (manager *SNatgatewayResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NatGatewayResourceInfo {
	rows := make([]api.NatGatewayResourceInfo, len(objs))
	natIds := make([]string, len(objs))
	for i := range objs {
		var base *SNatgatewayResourceBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil {
			natIds[i] = base.NatgatewayId
		}
	}
	nats := make(map[string]SNatGateway)
	err := db.FetchStandaloneObjectsByIds(NatGatewayManager, natIds, &nats)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	vpcList := make([]interface{}, len(rows))
	for i := range rows {
		rows[i] = api.NatGatewayResourceInfo{}
		if _, ok := nats[natIds[i]]; ok {
			rows[i].Natgateway = nats[natIds[i]].Name
			rows[i].VpcId = nats[natIds[i]].VpcId
		}
		vpcList[i] = &SVpcResourceBase{rows[i].VpcId}
	}

	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, vpcList, fields, isList)
	for i := range rows {
		rows[i].VpcResourceInfo = vpcRows[i]
	}

	return rows
}

func (manager *SNatgatewayResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NatGatewayFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.Natgateway) > 0 {
		natObj, _, err := ValidateNatGatewayResourceInput(userCred, query.NatGatewayResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateNatGatewayResourceInput")
		}
		q = q.Equals("natgateway_id", natObj.GetId())
	}

	subq := NatGatewayManager.Query("id").Snapshot()
	subq, err := manager.SVpcResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
	}
	if subq.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("natgateway_id"), subq.SubQuery()))
	}
	return q, nil
}

func (manager *SNatgatewayResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "natgateway":
		natQuery := NatGatewayManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(natQuery.Field("name", field))
		q = q.Join(natQuery, sqlchemy.Equals(q.Field("natgateway_id"), natQuery.Field("id")))
		q.GroupBy(natQuery.Field("name"))
		return q, nil
	}
	nats := NatGatewayManager.Query("id", "vpc_id").SubQuery()
	q = q.LeftJoin(nats, sqlchemy.Equals(q.Field("natgateway_id"), nats.Field("id")))
	q, err := manager.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SNatgatewayResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NatGatewayFilterListInput,
) (*sqlchemy.SQuery, error) {
	q, orders, fields := manager.GetOrderBySubQuery(q, userCred, query)
	if len(orders) > 0 {
		q = db.OrderByFields(q, orders, fields)
	}
	return q, nil
}

func (manager *SNatgatewayResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NatGatewayFilterListInput,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	natQ := NatGatewayManager.Query("id", "name")
	var orders []string
	var fields []sqlchemy.IQueryField

	if db.NeedOrderQuery(manager.SVpcResourceBaseManager.GetOrderByFields(query.VpcFilterListInput)) {
		var vpcOrders []string
		var vpcFields []sqlchemy.IQueryField
		natQ, vpcOrders, vpcFields = manager.SVpcResourceBaseManager.GetOrderBySubQuery(natQ, userCred, query.VpcFilterListInput)
		if len(vpcOrders) > 0 {
			orders = append(orders, vpcOrders...)
			fields = append(fields, vpcFields...)
		}
	}
	if db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		subq := natQ.SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("natgateway_id"), subq.Field("id")))
		if db.NeedOrderQuery([]string{query.OrderByNatgateway}) {
			orders = append(orders, query.OrderByNatgateway)
			fields = append(fields, subq.Field("name"))
		}
	}
	return q, orders, fields
}

func (manager *SNatgatewayResourceBaseManager) GetOrderByFields(query api.NatGatewayFilterListInput) []string {
	fields := make([]string, 0)
	vpcFields := manager.SVpcResourceBaseManager.GetOrderByFields(query.VpcFilterListInput)
	fields = append(fields, vpcFields...)
	fields = append(fields, query.OrderByNatgateway)
	return fields
}

/*
func (manager *SNatgatewayResourceBaseManager) FetchParentId(ctx context.Context, data jsonutils.JSONObject) string {
	parentId, _ := data.GetString("nategateway_id")
	return parentId
}

func (manager *SNatgatewayResourceBaseManager) FilterByParentId(q *sqlchemy.SQuery, parentId string) *sqlchemy.SQuery {
	if len(parentId) > 0 {
		q = q.Equals("nategateway_id", parentId)
	}
	return q
}
*/
