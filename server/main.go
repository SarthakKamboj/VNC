package main


// import "fmt"
import "sk_vnc/screen_capture"

/*
#include <math.h>
*/
import "C"

func main(){
	// fmt.Printf("Hello world! %f\n", C.M_PI)
	screen_capture.CaptureScreen()
}