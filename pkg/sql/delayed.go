// Copyright 2016 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

package sql

import (
	"context"

	"github.com/cockroachdb/cockroach/pkg/sql/catalog/colinfo"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
)

// delayedNode wraps a planNode in cases where the planNode
// constructor must be delayed during query execution (as opposed to
// SQL prepare) for resource tracking purposes.
type delayedNode struct {
	name        string
	columns     colinfo.ResultColumns
	constructor nodeConstructor
	plan        planNode
}

type nodeConstructor func(context.Context, *planner) (planNode, error)

func (d *delayedNode) Next(params runParams) (bool, error) { return d.plan.Next(params) }
func (d *delayedNode) Values() tree.Datums                 { return d.plan.Values() }

func (d *delayedNode) Close(ctx context.Context) {
	if d.plan != nil {
		d.plan.Close(ctx)
		d.plan = nil
	}
}

// startExec constructs the wrapped planNode now that execution is underway.
func (d *delayedNode) startExec(params runParams) error {
	if d.plan != nil {
		panic("wrapped plan should not yet exist")
	}

	plan, err := d.constructor(params.ctx, params.p)
	if err != nil {
		return err
	}
	d.plan = plan

	// Recursively invoke startExec on new plan. Normally, startExec doesn't
	// recurse - calling children is handled by the planNode walker. The reason
	// this won't suffice here is that the child of this node doesn't exist
	// until after startExec is invoked.
	return startExec(params, plan)
}
