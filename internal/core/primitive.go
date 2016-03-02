// Copyright 2016 The Vermeer Light Tools Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import ()

type Primitive interface {
	TraceRay(*RayData)
	VisRay(*RayData)
}
