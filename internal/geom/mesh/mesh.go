// Copyright 2016 The Vermeer Light Tools Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mesh

import (
	"errors"
	"github.com/jamiec7919/vermeer/core"
	m "github.com/jamiec7919/vermeer/math"
	"github.com/jamiec7919/vermeer/nodes"
	"github.com/jamiec7919/vermeer/qbvh"
)

// FaceGeom is the optimized triangular face structure.
// WARNING: Do not modify this without careful consideration! Designed to fit neatly in cache.
// 16 float32, 64bytes
type FaceGeom struct {
	V     [3]m.Vec3
	N     m.Vec3   // Not actually used in intersection code at moment..
	Vi    [3]int32 // reference the tex coords etc.
	MtlID int32    // Material Id
}

func (f *FaceGeom) setup() {
	f.N = m.Vec3Normalize(m.Vec3Cross(m.Vec3Sub(f.V[1], f.V[0]), m.Vec3Sub(f.V[2], f.V[0])))

	for i := range f.N {
		if f.N[i] == 0.0 {
			f.N[i] = 0
		}
	}
}

// Loader is implemented by mesh loaders.
type Loader interface {
	Load() (*Mesh, error)
	SetOption(opt string, v interface{}) error
}

// Mesh represents a triangular mesh.
type Mesh struct {
	Name            string
	Faces           []FaceGeom
	Vn              []m.Vec3
	Vuv             [][]m.Vec2
	nodes           []qbvh.Node
	faceindex       []int32 // Face indexes - used only with acceleration leaf structure, may contain duplicates
	bounds          m.BoundingBox
	RayBias         float32
	UseIndexedFaces bool
}

func (mesh *Mesh) calcVertexNormals() error {
	//log.Printf("CalcVertexNormals")
	if mesh.Vn == nil {
		if mesh.Vuv != nil {
			if mesh.Vuv[0] != nil {
				mesh.Vn = make([]m.Vec3, len(mesh.Vuv[0]))
			}
		} else {
			mesh.Vn = make([]m.Vec3, len(mesh.Faces)*3)
		}
	}

	for _, f := range mesh.Faces {
		mesh.Vn[f.Vi[0]] = m.Vec3Add(mesh.Vn[f.Vi[0]], f.N)
		mesh.Vn[f.Vi[1]] = m.Vec3Add(mesh.Vn[f.Vi[1]], f.N)
		mesh.Vn[f.Vi[2]] = m.Vec3Add(mesh.Vn[f.Vi[2]], f.N)
		//	log.Printf("%v %v %v %v %v %v %v", f.Vi[0], f.Vi[1], f.Vi[2], mesh.Vn[f.Vi[0]], mesh.Vn[f.Vi[1]], mesh.Vn[f.Vi[2]], f.N)
	}
	for i := range mesh.Vn {
		mesh.Vn[i] = m.Vec3Normalize(mesh.Vn[i])
	}

	return nil
}

// TraceRay implements core.Primitive.
func (mesh *Mesh) TraceRay(ray *core.RayData, sg *core.ShaderGlobals) int32 {
	if mesh.faceindex != nil {
		if mesh.RayBias == 0.0 {
			return mesh.traceRayAccelIndexed(ray, sg)
		}

		return mesh.traceRayAccelIndexedEpsilon(ray, sg)

	}

	if mesh.RayBias == 0.0 {
		return mesh.traceRayAccel(ray, sg)
	}

	return mesh.traceRayAccelEpsilon(ray, sg)

}

// VisRay implements core.Primitive.
func (mesh *Mesh) VisRay(ray *core.RayData) {
	if mesh.faceindex != nil {
		if mesh.RayBias == 0.0 {
			mesh.visRayAccelIndexed(ray)
		}
		mesh.visRayAccelIndexedEpsilon(ray)

	} else {
		if mesh.RayBias == 0.0 {
			mesh.visRayAccel(ray)
		}
		mesh.visRayAccelEpsilon(ray)

	}
}

// StaticMesh implements a static mesh (non file) node.
type StaticMesh struct {
	NodeName string
	Mesh     *Mesh
}

// Name implements core.Node.
func (mesh *StaticMesh) Name() string { return mesh.NodeName }

// PreRender implements core.Node.
func (mesh *StaticMesh) PreRender(rc *core.RenderContext) error {
	mesh.Mesh.initFaces()
	return mesh.Mesh.initAccel()
}

// PostRender implements core.Node.
func (mesh *StaticMesh) PostRender(rc *core.RenderContext) error { return nil }

// WorldBounds implements core.Primitive.
func (mesh *StaticMesh) WorldBounds() (out m.BoundingBox) {

	return mesh.Mesh.WorldBounds()
}

// Visible implements core.Primitive.
func (mesh *StaticMesh) Visible() bool { return true }

// TraceRay implements core.Primitive.
func (mesh *StaticMesh) TraceRay(ray *core.RayData, sg *core.ShaderGlobals) int32 {
	return mesh.Mesh.TraceRay(ray, sg)
}

// VisRay implements core.Primitive.
func (mesh *StaticMesh) VisRay(ray *core.RayData) {
	mesh.Mesh.VisRay(ray)
}

// Meshfile implements a mesh node loaded from a file.
type Meshfile struct {
	NodeName     string `node:"Name"`
	Filename     string
	RayBias      float32
	CalcNormals  bool
	IsVisible    bool
	MergeVertPos bool
	MergeTexVert bool
	Transform    m.Matrix4
	Loader       Loader
	mesh         *Mesh
}

// Name implements core.Node.
func (mesh *Meshfile) Name() string { return mesh.NodeName }

// PreRender implements core.Node.
func (mesh *Meshfile) PreRender(rc *core.RenderContext) error {

	for _, open := range loaders {
		loader, err := open(rc, mesh.Filename)

		if err == nil {

			mesh.Loader = loader
		}
	}

	if mesh.Loader == nil {
		return errors.New("No loader for mesh " + mesh.Filename)
	}
	mesh.Loader.SetOption("MergeVertPos", mesh.MergeVertPos)
	mesh.Loader.SetOption("MergeTexVert", mesh.MergeTexVert)

	msh, err := mesh.Loader.Load()

	if err != nil {
		return err
	}

	msh.RayBias = mesh.RayBias

	if mesh.NodeName == "mesh02" {
		trn := m.Matrix4Translate(-0.4, 0.3, 0.4)
		rot := m.Matrix4Rotate(m.Pi/2, 0, 1, 0)
		mesh.Transform = m.Matrix4Mul(trn, rot)
	}

	if !mesh.Transform.IsIdentity() {
		msh.ApplyTransform(mesh.Transform)
	}

	msh.initFaces()

	if mesh.CalcNormals {
		msh.calcVertexNormals()
	}
	/*
		if mesh.NodeName == "mesh02" {
			for i := range msh.Vn {
				msh.Vn[i] = m.Vec3Neg(msh.Vn[i])
			}

		}
	*/
	msh.Name = mesh.NodeName
	mesh.mesh = msh
	return mesh.mesh.initAccel()
}

// PostRender implements core.Node.
func (mesh *Meshfile) PostRender(rc *core.RenderContext) error { return nil }

// TraceRay implements core.Primitive.
func (mesh *Meshfile) TraceRay(ray *core.RayData, sg *core.ShaderGlobals) int32 {
	return mesh.mesh.TraceRay(ray, sg)
}

// Visible implements core.Primitive.
func (mesh *Meshfile) Visible() bool { return mesh.IsVisible }

// WorldBounds implements core.Primitive.
func (mesh *Meshfile) WorldBounds() (out m.BoundingBox) {
	return mesh.mesh.WorldBounds()
}

// VisRay implements core.Primitive.
func (mesh *Meshfile) VisRay(ray *core.RayData) {
	mesh.mesh.VisRay(ray)
}

// Transform applies the given transform to the vertices and recalculates normals.
func (mesh *Mesh) ApplyTransform(trn m.Matrix4) {
	for i := range mesh.Faces {
		mesh.Faces[i].V[0] = m.Matrix4MulPoint(trn, mesh.Faces[i].V[0])
		mesh.Faces[i].V[1] = m.Matrix4MulPoint(trn, mesh.Faces[i].V[1])
		mesh.Faces[i].V[2] = m.Matrix4MulPoint(trn, mesh.Faces[i].V[2])
		//mesh.Faces[i].N = m.Matrix4MulVec(trn, mesh.Faces[i].N)
	}

	for i := range mesh.Vn {
		mesh.Vn[i] = m.Matrix4MulVec(trn, mesh.Vn[i])
	}

}

// WorldBounds implements core.Primitive.
func (mesh *Mesh) WorldBounds() (out m.BoundingBox) {
	out.Reset()

	for i := range mesh.Faces {
		for k := range mesh.Faces[i].V {
			out.GrowVec3(mesh.Faces[i].V[k])
		}
	}
	return
}

func calcBox(verts []m.Vec3) (bounds m.BoundingBox) {
	bounds.Reset()

	for i := range verts {
		bounds.GrowVec3(verts[i])
	}
	return
}

/*
Change this for early split clipping, check each triangle BB for surface area, if greater than a maximum then
split along longest axis, and add two new bounding boxes+indices to original triangle.  Need to reintroduce
indices or copy triangles. Maybe use median BB surface area for mesh as maximum.
*/
func trisplit(verts []m.Vec3, idx int32, indexes *[]int32, boxes *[]m.BoundingBox, centroids *[]m.Vec3) {
	var stack [][]m.Vec3
	stack = append(stack, verts)
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		var box m.BoundingBox
		box.Reset()

		if len(top) < 1 {
			panic("<1")
		}
		for i := range top {
			box.GrowVec3(top[i])
		}

		if false /*box.SurfaceArea() > 50000000*/ {
			axis := box.MaxDim()

			d := box.Centroid()[axis]

			//log.Printf("SA: %v %v %v", box.SurfaceArea(), axis, d)
			primng := qbvh.ClipLeft(d, axis, top)
			//log.Printf("%v", primng)
			primpv := qbvh.ClipRight(d, axis, top)
			stack = append(stack, primng)
			//log.Printf("%v", primpv)
			stack = append(stack, primpv)
		} else {
			*indexes = append(*indexes, idx)
			*boxes = append(*boxes, box)

			var centroid m.Vec3
			for i := range top {
				centroid = m.Vec3Add(centroid, top[i])
			}

			centroid = m.Vec3Scale(1.0/float32(len(top)), centroid)
			*centroids = append(*centroids, centroid)

			//log.Printf("Add: %v %v %v", idx, box, centroid)
		}
	}
}

func (mesh *Mesh) initFaces() {
	for face := range mesh.Faces {
		mesh.Faces[face].setup()
		//log.Printf("%v %v ", m.Faces[face].N, m.Faces[face].MtlID)
	}

}

func (mesh *Mesh) initAccel() error {

	boxes := make([]m.BoundingBox, 0, len(mesh.Faces))
	centroids := make([]m.Vec3, 0, len(mesh.Faces))
	indxs := make([]int32, 0, len(mesh.Faces))

	for i := range mesh.Faces {
		verts := make([]m.Vec3, 3)
		verts[0] = mesh.Faces[i].V[0]
		verts[1] = mesh.Faces[i].V[1]
		verts[2] = mesh.Faces[i].V[2]
		trisplit(verts, int32(i), &indxs, &boxes, &centroids)
		/*box := m.BoundingBox{}
		box.Reset()
		box.Grow(m.Faces[i].V[0][0], m.Faces[i].V[0][1], m.Faces[i].V[0][2])
		box.Grow(m.Faces[i].V[1][0], m.Faces[i].V[1][1], m.Faces[i].V[1][2])
		box.Grow(m.Faces[i].V[2][0], m.Faces[i].V[2][1], m.Faces[i].V[2][2])
		centroid := m.Vec3{}
		centroid[0] = (m.Faces[i].V[0][0] + m.Faces[i].V[1][0] + m.Faces[i].V[2][0]) / 3
		centroid[1] = (m.Faces[i].V[0][1] + m.Faces[i].V[1][1] + m.Faces[i].V[2][1]) / 3
		centroid[2] = (m.Faces[i].V[0][2] + m.Faces[i].V[1][2] + m.Faces[i].V[2][2]) / 3
		indxs = append(indxs, int32(i))
		boxes = append(boxes, box)
		centroids = append(centroids, centroid)
		*/
	}

	//for i := range boxes {
	//log.Printf("%v", boxes[i].SurfaceArea())
	//	}
	//panic("stop")
	nodes, bounds := qbvh.BuildAccel(boxes, centroids, indxs, 16)

	mesh.nodes = nodes
	mesh.bounds = bounds

	if !mesh.UseIndexedFaces {
		newfaces := make([]FaceGeom, len(mesh.Faces))

		// rearrange faces to match index structure of nodes
		for i := range indxs {
			newfaces[i] = mesh.Faces[indxs[i]]
		}

		mesh.Faces = newfaces
	} else {
		mesh.faceindex = indxs

	}

	return nil
}

var loaders = map[string]func(rc *core.RenderContext, filename string) (Loader, error){}

// RegisterLoader registers the given mesh file loader.
func RegisterLoader(name string, open func(rc *core.RenderContext, filename string) (Loader, error)) {
	loaders[name] = open
}

func create() (core.Node, error) {
	mfile := Meshfile{Transform: m.Matrix4Identity, IsVisible: true}

	return &mfile, nil
}

func init() {
	nodes.Register("Meshfile", create)
}
