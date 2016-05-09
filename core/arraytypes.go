// Copyright 2016 The Vermeer Light Tools Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	m "github.com/jamiec7919/vermeer/math"
)

/*
 These wrap the idea of arrays of elements (e.g. points, vec2's or matrices) for motion keys.
 storage is simply an array of the appropriate type, of length MotionKeys*ElemsPerKey (for types
 where that makes sense).
*/

type PointArray struct {
	MotionKeys  int // Number of motion keys
	ElemsPerKey int // Number of elements per key
	Elems       []m.Vec3
}

type Vec2Array struct {
	MotionKeys  int // Number of motion keys
	ElemsPerKey int // Number of elements per key
	Elems       []m.Vec2
}

type MatrixArray struct {
	MotionKeys int // Number of motion keys
	Elems      []m.Matrix4
}
