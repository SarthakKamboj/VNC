package main

import "sk_vnc/mac_capture"

import "fmt"

/*
#include <math.h>
*/
import "C"
import "unsafe"

func main(){
	fmt.Printf("Hello world! %f\n", C.M_PI)
	var scConfig unsafe.Pointer = mac_capture.CreateScStreamConfiguration()
	fmt.Printf("%x", scConfig)
}