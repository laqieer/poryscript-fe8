#pragma once

#include "global.h"
#include "bmunit.h"
#include "event.h"
#include "eventinfo.h"
#include "eventcall.h"
#include "EAstdlib.h"
#include "constants/characters.h"
#include "constants/backgrounds.h"
#include "constants/items.h"
#include "constants/songs.h"
#include "constants/chapters.h"

CONST_DATA EventListScr EventScr_Sample_BeginningScene[] = {
	CALL(EventScr_Sample_Intro)
	MUSC(SONG_RAID)
	LOAD1(1, UnitDef_Event_SampleAlly)
	ENUN
	MOVE(0x18, CHARACTER_SETH, 4, 4)
	ENUN
	FlashCursor(CHARACTER_SETH, 60)
	Text(0x90E)
	CHECK_EVENTID(EVFLAG_TMP(8))
	BNE(0x2, EVT_SLOT_C, EVT_SLOT_0)
	Text(0x910)
LABEL(0x1)
	ENUT(0x8)
	NoFade
	ENDA
LABEL(0x2)
	ASMC(BmGuideTextSetAllGreen)
	Text(0x90F)
	GOTO(0x1)
};

CONST_DATA EventListScr EventScr_Sample_Intro[] = {
	FADI(16)
	FlashCursor(CHARACTER_EIRIKA, 60)
	Text(0x903)
	ENDA
};
