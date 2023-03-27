#pragma once

#include <./include/openh264/codec_api.h>

#ifdef __cplusplus
extern "C" {
#endif

int EncodeFrame(ISVCEncoder *encoder, SSourcePicture *pic, SFrameBSInfo *bsInfo);
int EncoderInit(ISVCEncoder *encoder, const SEncParamBase* pParam);

#ifdef __cplusplus
}
#endif
