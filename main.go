package main

//#cgo CFLAGS: -I${SRCDIR}/include
//#cgo LDFLAGS: -L${SRCDIR}/lib -lopenh264
//#include <openh264/codec_api.h>
//#include <string.h>
//#include <stdint.h>
//#include <stdlib.h>
//#include "lib.hpp"
import "C"
import (
	"fmt"
	"os"
	"time"
	"unsafe"
	_ "unsafe"

	"gocv.io/x/gocv"
)

func main() {
	var mp4Filename = "./1.mp4"
	vc, err := gocv.VideoCaptureFile(mp4Filename)
	if err != nil {
		panic(err)
	}
	defer vc.Close()

	fps := vc.Get(gocv.VideoCaptureFPS)
	width := int(vc.Get(gocv.VideoCaptureFrameWidth))
	height := int(vc.Get(gocv.VideoCaptureFrameHeight))
	bitRate := int(vc.Get(gocv.VideoCaptureBitrate))
	//rcMode := int(vc.Get(gocv.VideoCaptureMode))

	t := time.Now().Format("2006_01_02_15_04_05")
	path := "./" + t
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		panic(err)
	}

	var svcEncoder *C.ISVCEncoder
	errCode := C.WelsCreateSVCEncoder(&svcEncoder)
	if errCode != 0 {
		return
	}
	defer C.WelsDestroySVCEncoder(svcEncoder)

	var encParam C.SEncParamBase
	encParam.iPicWidth = C.int32_t(width)
	encParam.iPicHeight = C.int32_t(height)
	encParam.iTargetBitrate = C.int32_t(bitRate)
	encParam.iRCMode = C.RC_OFF_MODE
	encParam.fMaxFrameRate = C.float(fps)
	rv := C.EncoderInit(svcEncoder, &encParam)
	if rv != 0 {
		panic("EncoderInit failed")
	}

	writer, err := gocv.VideoWriterFile("h264.mp4", "avc1", fps, width, height, true)
	if err != nil {
		panic(err)
	}
	defer writer.Close()

	for finished := false; !finished; {
		finished = encode(vc, height, width, svcEncoder, writer)
	}
}

func encode(vc *gocv.VideoCapture, height int, width int, svcEncoder *C.ISVCEncoder, writer *gocv.VideoWriter) bool {
	mat := gocv.NewMat()
	defer mat.Close()
	if ok := vc.Read(&mat); !ok {
		fmt.Println("cannot read mat")
		return true
	}
	if mat.Empty() {
		fmt.Println("mat is empty")
		return false
	}
	yuv, err := mat.ToImageYUV()
	if err != nil {
		fmt.Println("ToImageYUV failed.", err.Error())
		return false
	}
	yuvMat := gocv.NewMatWithSize(height, width, mat.Type())
	defer yuvMat.Close()
	gocv.CvtColor(mat, &yuvMat, gocv.ColorBGRToYUV)

	var srcPic C.SSourcePicture
	srcPic.iPicWidth = C.int(width)
	srcPic.iPicHeight = C.int(height)
	srcPic.iColorFormat = C.int(C.videoFormatI420)
	srcPic.uiTimeStamp = C.longlong(time.Now().UnixNano())
	srcPic.iStride[0] = C.int(yuv.YStride)
	srcPic.iStride[1] = C.int(yuv.CStride)
	srcPic.iStride[2] = C.int(yuv.CStride)
	ptrUint8, err := yuvMat.DataPtrUint8()
	if err != nil {
		fmt.Println("DataPtrUint8 failed.", err.Error())
		return false
	}
	srcPic.pData[0] = (*C.uchar)(unsafe.Pointer(&ptrUint8[0]))
	srcPic.pData[1] = (*C.uchar)(unsafe.Pointer(&ptrUint8[width*height]))
	srcPic.pData[2] = (*C.uchar)(unsafe.Pointer(&ptrUint8[width*height+width*height>>2]))

	var frameBSInfo C.SFrameBSInfo
	rv := C.EncodeFrame(svcEncoder, &srcPic, &frameBSInfo)
	if rv == VideoFrameTypeInvalid {
		fmt.Println("EncodeFrame failed")
		return false
	} else if rv != VideoFrameTypeSkip {

		// 使用C.SFrameBSInfo中的数据生成视频文件
		d := make([]byte, 0)
		for i := 0; i < int(frameBSInfo.iLayerNum); i++ {
			layer := frameBSInfo.sLayerInfo[i]
			var nalLen = 0
			for j := 0; j < int(layer.iNalCount); j++ {
				nalLen += int(*layer.pNalLengthInByte)
			}
			data := C.GoBytes(unsafe.Pointer(layer.pBsBuf), C.int(nalLen))
			d = append(d, data...)
		}

		imDecode, err := gocv.IMDecode(d, gocv.IMReadAnyColor)
		if err != nil {
			fmt.Println("imDecode failed.", err.Error())
			return false
		}
		err = writer.Write(imDecode)
		if err != nil {
			fmt.Println("writer.Write failed.", err.Error())
			return false
		}
	}
	return false
}

const (
	VideoFrameTypeInvalid = 0x0
	VideoFrameTypeIDR     = 0x1
	VideoFrameTypeI       = 0x2
	VideoFrameTypeP       = 0x3
	VideoFrameTypeSkip    = 0x4
	VideoFrameTypeIPMixed = 0x5
)
