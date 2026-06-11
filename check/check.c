/* Macro-resolution / compile check for the generated FE8 event header.
 *
 * The generated header already #includes the decomp's event headers
 * (global.h, EAstdlib.h, etc.), so this just pulls it in. Preprocessing/compiling
 * it against the read-only decomp headers confirms every emitted macro/symbol
 * resolves. See the README and check.sh for the exact commands used.
 *
 * The forward declarations below stand in for the project-specific symbols our
 * standalone sample references (another event script, a unit definition, and an
 * ASM-called function); in a real chapter these live elsewhere in the decomp. */
#include "global.h"
#include "bmunit.h"
#include "event.h"
#include "EAstdlib.h"

extern EventListScr EventScr_Sample_Intro[];
extern struct UnitDefinition UnitDef_Event_SampleAlly[];
void BmGuideTextSetAllGreen(void);

#include "sample.h"
