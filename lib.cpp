#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "lib.hpp"

int EncodeFrame(ISVCEncoder *encoder, SSourcePicture *pic, SFrameBSInfo *bsInfo) {
	int ret = encoder->EncodeFrame(pic, bsInfo);
	return ret;
}

int EncoderInit(ISVCEncoder *encoder, const SEncParamBase* pParam) {
    int ret = encoder->Initialize(pParam);
    return ret;
}