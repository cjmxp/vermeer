// Copyright 2016 The Vermeer Light Tools Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"github.com/jamiec7919/vermeer/colour"
	m "github.com/jamiec7919/vermeer/math"
	"github.com/jamiec7919/vermeer/qbvh"
	"sync/atomic"
)

// Scene represents the set of primitives and lights and
// world acceleration structure.
//
// Deprecated: Scene will become an interface.
type Scene struct {
	prims  []Primitive
	nodes  []qbvh.Node
	bounds m.BoundingBox

	lights []Light
}

var grc *RenderContext

// ScreenSample is returned by Trace.
type ScreenSample struct {
	Colour  colour.RGB
	Opacity colour.RGB
	Alpha   float32
	Point   m.Vec3
	Z       float64
	ElemID  uint32
	Prim    Primitive
}

// TraceProbe intersects ray with the scene and sets up the globals sg with the first intersection.
// rays of type RayShadow will early-out and not necessarily return the first intersection.
// Returns true if any intersection or false for none.
func TraceProbe(ray *RayData, sg *ShaderGlobals) bool {

	atomic.AddUint64(&rayCount, 1)

	if ray.Type&RayShadow != 0 {
		grc.scene.visRayAccel(ray)
		atomic.AddUint64(&shadowRays, 1)
		return !ray.IsVis()
	}

	mtlid := grc.scene.traceRayAccel(ray, sg)

	if mtlid != -1 {

		mtl := GetMaterial(mtlid)

		sg.Shader = mtl
		sg.N = m.Vec3Normalize(sg.N)
		sg.Ns = m.Vec3Normalize(sg.Ns)
		return true
	}

	return false
}

// Trace intersects ray with the scene and evaluates the shader at the first intersection. The
// result is returned in the samp struct.
// Returns true if any intersection or false for none.
func Trace(ray *RayData, samp *ScreenSample) bool {
	sg := &ShaderGlobals{
		Ro:     ray.Ray.P,
		Rd:     ray.Ray.D,
		Prim:   ray.Result.Prim,
		ElemID: ray.Result.ElemID,
		Depth:  ray.Level,
		rnd:    ray.rnd,
		Lambda: ray.Lambda,
		Time:   ray.Time,
	}

	if TraceProbe(ray, sg) {
		if sg.Shader == nil { // can't do much with no material
			return false
		}

		sg.Shader.Eval(sg)

		if samp != nil {
			samp.Colour = sg.OutRGB
			samp.Point = sg.Ro
			samp.ElemID = sg.ElemID
			samp.Prim = sg.Prim
		}

		return true
	}
	return false
}

func (scene *Scene) initAccel() error {
	boxes := make([]m.BoundingBox, 0, len(scene.prims))
	indices := make([]int32, 0, len(scene.prims))
	centroids := make([]m.Vec3, 0, len(scene.prims))

	for i := range scene.prims {
		if !scene.prims[i].Visible() {
			continue
		}
		box := scene.prims[i].WorldBounds()
		boxes = append(boxes, box)
		indices = append(indices, int32(i))
		centroids = append(centroids, box.Centroid())
	}

	nodes, bounds := qbvh.BuildAccel(boxes, centroids, indices, 1)

	scene.nodes = nodes

	// Rearrange (visible) primitive array to match leaf structure
	nprims := make([]Primitive, len(indices))

	for i := range indices {
		nprims[i] = scene.prims[indices[i]]
	}

	scene.prims = nprims
	scene.bounds = bounds

	return nil
}
