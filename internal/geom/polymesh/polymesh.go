// Copyright 2016 The Vermeer Light Tools Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package polymesh

import (
	"github.com/jamiec7919/vermeer/core"
	m "github.com/jamiec7919/vermeer/math"
	"github.com/jamiec7919/vermeer/nodes"
	"github.com/jamiec7919/vermeer/qbvh"
)

type PolyMesh struct {
	NodeName string `node:"Name"`
	RayBias  float32

	Verts     core.PointArray
	PolyCount []int32
	FaceIdx   []int32

	Material     string
	ModelToWorld core.MatrixArray
	CalcNormals  bool
	IsVisible    bool

	UV    core.Vec2Array
	UVIdx []int32

	Normals   core.Vec3Array
	NormalIdx []int32

	facecount     int      // Number of faces
	idxp          []uint32 // Triangle Face indexes (position)
	vertidxstride int      // 3 or 4 if including material ids
	uvtriidx      []uint32 // triangulated UV indexes
	normalidx     []uint32

	accel struct {
		mqbvh qbvh.MotionQBVH
		qbvh  []qbvh.Node
		idx   []int32 // Face indexes
	}

	mtlid int32

	bounds m.BoundingBox
}

func (mesh *PolyMesh) Name() string { return mesh.NodeName }
func (mesh *PolyMesh) PreRender(rc *core.RenderContext) error {
	if err := mesh.init(); err != nil {
		return err
	}

	mesh.facecount = len(mesh.idxp) / 3
	mesh.vertidxstride = 3

	mesh.mtlid = core.GetMaterialId(mesh.Material)

	return mesh.initAccel()
}
func (mesh *PolyMesh) PostRender(rc *core.RenderContext) error { return nil }

func (mesh *PolyMesh) WorldBounds() (out m.BoundingBox) {

	return mesh.bounds
}

func (mesh *PolyMesh) Visible() bool { return true }

func create() (core.Node, error) {
	mfile := PolyMesh{IsVisible: true}

	return &mfile, nil
}

func init() {
	nodes.Register("PolyMesh", create)
}
