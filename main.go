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
		return
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
		return
	}

	//f, _ := os.Open("./h264.mp4")

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
		return
	}
	defer writer.Close()

	var index = 0
	for ; ; index++ {
		mat := gocv.NewMat()
		//defer mat.Close()
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
		yuvMat := gocv.NewMatWithSize(height*3/2, width, mat.Type())
		gocv.CvtColor(mat, &yuvMat, gocv.ColorBGRToYUV)

		var srcPic C.SSourcePicture
		srcPic.iPicWidth = C.int(width)
		srcPic.iPicHeight = C.int(height)
		srcPic.iColorFormat = C.int(C.videoFormatI420)
		srcPic.uiTimeStamp = C.longlong(time.Now().UnixNano())
		srcPic.iStride[0] = C.int(yuv.YStride)
		srcPic.iStride[1] = C.int(yuv.CStride)
		srcPic.iStride[2] = C.int(yuv.CStride)
		//bytes := yuvMat.ToBytes()
		//ptrUint8, err := yuvMat.DataPtrUint8()
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
				//imDecode, err := gocv.IMDecode(data, gocv.IMReadUnchanged)
				//if err != nil {
				//	panic(err)
				//}
				//err = writer.Write(imDecode)
				//if err != nil {
				//	panic(err)
				//}
			}
		}
		//f.Write(d)
		imDecode, err := gocv.IMDecode(d, gocv.IMReadAnyColor|gocv.IMReadAnyDepth)
		if err != nil {
			panic(err)
		}
		fmt.Println(imDecode.Rows())
		err = writer.Write(imDecode)
		if err != nil {
			panic(err)
		}
	}
}

func getYUVStride(yuvImage gocv.Mat) (int, int, int) {
	yStride := yuvImage.Step()
	var uStride, vStride int

	// 如果图像不是标准的YUV 4:2:0格式，则需要根据实际情况进行调整
	t := yuvImage.Type()
	if t == gocv.MatTypeCV8UC2 || t == gocv.MatTypeCV16UC2 || t == gocv.MatTypeCV16SC2 {
		// YUV 4:2:2格式，U和V通道的stride是Y通道stride的1/2
		uStride = yStride / 2
		vStride = yStride / 2
	} else if t == gocv.MatTypeCV8UC3 || t == gocv.MatTypeCV16UC3 || t == gocv.MatTypeCV16SC3 {
		// YUV 4:4:4格式，U和V通道的stride等于Y通道stride
		uStride = yStride
		vStride = yStride
	} else if t == gocv.MatTypeCV8UC4 || t == gocv.MatTypeCV16UC4 || t == gocv.MatTypeCV16SC4 {
		// YUV 4:2:0格式，U和V通道的stride是Y通道stride的1/2
		yStride /= 2
		uStride = yStride / 2
		vStride = yStride / 2
	}
	return yStride, uStride, vStride
}

func parseSSourceToH264Mat(inPic C.SSourcePicture) *gocv.Mat {
	// Convert SSourcePicture to gocv.Mat
	return sSourcePictureToH264Mat(inPic)
}

func sSourcePictureToH264Mat(inPic C.SSourcePicture) *gocv.Mat {
	// 创建一个名为mat的Mat对象
	mat := gocv.NewMatWithSize(int(inPic.iPicHeight)*3/2, int(inPic.iPicWidth), gocv.MatTypeCV8UC1)

	// 通过循环遍历每个通道的数据，并将它们复制到mat对象中
	for i := 0; i < 3; i++ {
		ptrUint8, err := mat.DataPtrUint8()
		if err != nil {
			fmt.Println("DataPtrUint8 failed:", err)
			return nil
		}
		if inPic.iStride[i] == inPic.iPicWidth {
			// 如果stride等于宽度，则直接复制整个通道的数据
			copy(ptrUint8[(i*int(inPic.iPicHeight)):(i*int(inPic.iPicHeight)+int(inPic.iPicHeight))], (*[1 << 30]byte)(unsafe.Pointer(inPic.pData[i]))[:])
		} else {
			// 如果stride不等于宽度，则按行复制数据
			for j := 0; j < int(inPic.iPicHeight); j++ {
				src := (*[1 << 30]byte)(unsafe.Pointer(uintptr(unsafe.Pointer(inPic.pData[i])) + uintptr(j*int(inPic.iStride[i]))))
				dst := ptrUint8[i*int(inPic.iPicHeight)+j : i*int(inPic.iPicHeight)+j+1]
				copy(dst, src[0:inPic.iPicWidth])
			}
		}
	}

	return &mat
}
