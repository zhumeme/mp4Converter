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

	for {
		mat := gocv.NewMat()
		if ok := vc.Read(&mat); !ok {
			return
		}
		if mat.Empty() {
			mat.Close()
			continue
		}
		yuv, err := mat.ToImageYUV()
		if err != nil {
			panic(err)
		}
		yuvMat := gocv.NewMatWithSize(height, width, mat.Type())
		gocv.CvtColor(mat, &yuvMat, gocv.ColorBGRToYUV)

		var srcPic C.SSourcePicture
		srcPic.iPicWidth = C.int(width)
		srcPic.iPicHeight = C.int(height)
		srcPic.iColorFormat = C.int(C.videoFormatI420)
		srcPic.uiTimeStamp = C.longlong(time.Now().UnixNano())
		srcPic.iStride[0] = C.int(yuv.YStride)
		srcPic.iStride[1] = C.int(yuv.CStride)
		srcPic.iStride[2] = C.int(yuv.CStride)
		srcPic.pData[0] = (*C.uchar)(C.CBytes(yuv.Y))
		srcPic.pData[1] = (*C.uchar)(C.CBytes(yuv.Cb))
		srcPic.pData[2] = (*C.uchar)(C.CBytes(yuv.Cr))

		var frameBSInfo C.SFrameBSInfo
		rv = C.EncodeFrame(svcEncoder, &srcPic, &frameBSInfo)
		if rv != 0 {
			fmt.Println("EncodeFrame failed")
			panic("EncodeFrame failed")
		}

		// 使用C.SFrameBSInfo中的数据生成视频文件
		d := make([]byte, 0)
		for i := 0; i < int(frameBSInfo.iLayerNum); i++ {
			layer := frameBSInfo.sLayerInfo[i]
			for j := 0; j < int(layer.iNalCount); j++ {
				nalLen := int(*layer.pNalLengthInByte)
				data := C.GoBytes(unsafe.Pointer(layer.pBsBuf), C.int(nalLen))
				d = append(d, data...)
				//imDecode, err := gocv.IMDecode(data, gocv.IMReadAnyColor)
				//if err != nil {
				//	panic(err)
				//}
				//err = writer.Write(imDecode)
				//if err != nil {
				//	panic(err)
				//}
			}
		}

		imDecode, err := gocv.IMDecode(d, gocv.IMReadAnyColor)
		if err != nil {
			panic(err)
		}
		err = writer.Write(imDecode)
		if err != nil {
			panic(err)
		}
	}
}
