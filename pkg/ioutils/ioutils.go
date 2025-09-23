package ioutils

/*
 Copyright (c) 2025 Rhys Bryant

 This program is free software: you can redistribute it and/or modify
 it under the terms of the GNU General Public License as published by
 the Free Software Foundation, either version 3 of the License, or
 (at your option) any later version.

 This program is distributed in the hope that it will be useful,
 but WITHOUT ANY WARRANTY; without even the implied warranty of
 MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 GNU General Public License for more details.

 You should have received a copy of the GNU General Public License
 along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"io"
	"sync"
	"sync/atomic"
)

// ByoDirectionalCopy copies data bidirectionally between two io.ReadWriteCloser streams.
func ByoDirectionalCopy(dst io.ReadWriteCloser, src io.ReadWriteCloser) error {

	wait := sync.WaitGroup{}
	wait.Add(2)
	var errForReturn error

	// when one fails cause the other direcion to close too
	// one of the goroutines will likely return a closed connectionn an error
	// ignore the second error
	firstClose := atomic.Bool{}

	// Copy from src to dst
	go func() {
		_, err := io.Copy(dst, src)
		if !firstClose.Load() && err != nil && err != io.EOF {
			errForReturn = err
		}
		dst.Close()
		firstClose.Store(true)
		wait.Done()
	}()

	// Copy from dst to src
	go func() {
		_, err := io.Copy(src, dst)
		if !firstClose.Load() && err != nil && err != io.EOF {
			errForReturn = err
		}
		src.Close()
		firstClose.Store(true)
		wait.Done()
	}()

	// Wait for both directions to finish
	wait.Wait()
	return errForReturn
}
