/*
 * lcse_hook.dll v5.2 - Single-byte accents with sign-extension fix
 *
 * The LCSE engine passes single-byte chars via signed char, causing
 * sign-extension when cast to UINT for GetGlyphOutlineA:
 *   0xA1 (char = -95) -> UINT 0xFFFFFFA1
 * We must mask to (uChar & 0xFF) before comparing.
 *
 * Cross-compile:
 *   i686-w64-mingw32-gcc -shared -o lcse_hook.dll lcse_hook.c -lgdi32
 */
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdio.h>

typedef HFONT (WINAPI *pfnCreateFontIndirectA)(const LOGFONTA*);
typedef DWORD (WINAPI *pfnGetGlyphOutlineA)(HDC, UINT, UINT,
              LPGLYPHMETRICS, DWORD, LPVOID, const MAT2*);

static pfnCreateFontIndirectA g_origCreateFontIndirectA = NULL;
static pfnGetGlyphOutlineA    g_origGetGlyphOutlineA = NULL;

static char  g_fontName[LF_FACESIZE] = "";
static BOOL  g_fontOverride = FALSE;
static int   g_debugLog = 0;
static FILE *g_logFile = NULL;

/* ── Single-byte accent mapping ── */
#define ACCENT_BASE 0xA1
#define ACCENT_COUNT 13

static const WCHAR g_accentUnicode[ACCENT_COUNT] = {
    0x00E9,  /* 0xA1 = é */
    0x00E8,  /* 0xA2 = è */
    0x00E7,  /* 0xA3 = ç */
    0x00E0,  /* 0xA4 = à */
    0x00E2,  /* 0xA5 = â */
    0x00FB,  /* 0xA6 = û */
    0x00F4,  /* 0xA7 = ô */
    0x00EA,  /* 0xA8 = ê */
    0x00EE,  /* 0xA9 = î */
    0x00F9,  /* 0xAA = ù */
    0x00EB,  /* 0xAB = ë */
    0x00EF,  /* 0xAC = ï */
    0x00FC,  /* 0xAD = ü */
};

static void LogMsg(const char *fmt, ...)
{
    if (!g_debugLog || !g_logFile) return;
    va_list ap;
    va_start(ap, fmt);
    vfprintf(g_logFile, fmt, ap);
    va_end(ap);
    fflush(g_logFile);
}

static HFONT WINAPI Hook_CreateFontIndirectA(const LOGFONTA *lplf)
{
    if (g_fontOverride && lplf != NULL) {
        LOGFONTA lf = *lplf;
        LogMsg("[FONT] '%s' h=%d cs=%d -> ",
               lplf->lfFaceName, lplf->lfHeight, lplf->lfCharSet);
        strncpy(lf.lfFaceName, g_fontName, LF_FACESIZE - 1);
        lf.lfFaceName[LF_FACESIZE - 1] = '\0';
        if (lf.lfCharSet == DEFAULT_CHARSET || lf.lfCharSet == SHIFTJIS_CHARSET)
            lf.lfCharSet = SHIFTJIS_CHARSET;
        LogMsg("'%s' cs=%d\n", lf.lfFaceName, lf.lfCharSet);
        return g_origCreateFontIndirectA(&lf);
    }
    return g_origCreateFontIndirectA(lplf);
}

static DWORD WINAPI Hook_GetGlyphOutlineA(HDC hdc, UINT uChar, UINT fuFormat,
                                           LPGLYPHMETRICS lpgm, DWORD cjBuffer,
                                           LPVOID pvBuffer, const MAT2 *lpmat2)
{
    /*
     * KEY FIX: The engine sign-extends single bytes:
     *   0xA1 arrives as 0xFFFFFFA1
     *   0x41 arrives as 0x00000041
     * We mask to 8 bits for single-byte detection.
     * Double-byte SJIS codes (like 0x8163) have the lead byte in bits 15-8
     * and trail byte in bits 7-0, so they won't be affected by the mask
     * because we only check the masked value when the original matches
     * the sign-extension pattern.
     */
    UINT byte8 = uChar & 0xFF;

    /* Detect sign-extended single byte: 0xFFFFFFxx where xx >= 0x80 */
    BOOL isSignExtended = ((uChar & 0xFFFFFF00) == 0xFFFFFF00);

    /* Also handle non-sign-extended case (just in case) */
    BOOL isPlainHighByte = (uChar >= ACCENT_BASE && uChar <= 0xFF);

    if ((isSignExtended || isPlainHighByte) &&
        byte8 >= ACCENT_BASE && byte8 < ACCENT_BASE + ACCENT_COUNT)
    {
        WCHAR uc = g_accentUnicode[byte8 - ACCENT_BASE];
        DWORD result = GetGlyphOutlineW(hdc, (UINT)uc, fuFormat,
                                         lpgm, cjBuffer, pvBuffer, lpmat2);
        LogMsg("[ACCENT] 0x%08X -> byte 0x%02X -> U+%04X gmIncX=%d\n",
               uChar, byte8, (unsigned)uc,
               lpgm ? (int)lpgm->gmCellIncX : -1);
        return result;
    }

    /* Pass through everything else */
    return g_origGetGlyphOutlineA(hdc, uChar, fuFormat,
                                   lpgm, cjBuffer, pvBuffer, lpmat2);
}

/* ── IAT Patching ── */

static BOOL PatchIAT(HMODULE hModule, const char *dllName,
                     const char *funcName, void *hookFunc, void **origFunc)
{
    PIMAGE_DOS_HEADER dosHeader = (PIMAGE_DOS_HEADER)hModule;
    if (dosHeader->e_magic != IMAGE_DOS_SIGNATURE) return FALSE;
    PIMAGE_NT_HEADERS ntHeaders = (PIMAGE_NT_HEADERS)(
        (BYTE*)hModule + dosHeader->e_lfanew);
    if (ntHeaders->Signature != IMAGE_NT_SIGNATURE) return FALSE;
    DWORD importRVA = ntHeaders->OptionalHeader
        .DataDirectory[IMAGE_DIRECTORY_ENTRY_IMPORT].VirtualAddress;
    if (importRVA == 0) return FALSE;
    PIMAGE_IMPORT_DESCRIPTOR imports = (PIMAGE_IMPORT_DESCRIPTOR)(
        (BYTE*)hModule + importRVA);
    for (; imports->Name; imports++) {
        const char *name = (const char*)((BYTE*)hModule + imports->Name);
        if (_stricmp(name, dllName) != 0) continue;
        PIMAGE_THUNK_DATA origThunk = (PIMAGE_THUNK_DATA)(
            (BYTE*)hModule + imports->OriginalFirstThunk);
        PIMAGE_THUNK_DATA firstThunk = (PIMAGE_THUNK_DATA)(
            (BYTE*)hModule + imports->FirstThunk);
        for (; origThunk->u1.AddressOfData; origThunk++, firstThunk++) {
            if (origThunk->u1.Ordinal & IMAGE_ORDINAL_FLAG) continue;
            PIMAGE_IMPORT_BY_NAME importByName = (PIMAGE_IMPORT_BY_NAME)(
                (BYTE*)hModule + origThunk->u1.AddressOfData);
            if (strcmp((const char*)importByName->Name, funcName) == 0) {
                DWORD oldProtect;
                VirtualProtect(&firstThunk->u1.Function, sizeof(void*),
                               PAGE_READWRITE, &oldProtect);
                *origFunc = (void*)firstThunk->u1.Function;
                firstThunk->u1.Function = (ULONG_PTR)hookFunc;
                VirtualProtect(&firstThunk->u1.Function, sizeof(void*),
                               oldProtect, &oldProtect);
                LogMsg("[HOOK] Patched %s!%s OK\n", dllName, funcName);
                return TRUE;
            }
        }
    }
    LogMsg("[HOOK] FAILED %s!%s\n", dllName, funcName);
    return FALSE;
}

/* ── Config + Entry ── */

static void LoadConfig(void)
{
    char iniPath[MAX_PATH];
    GetModuleFileNameA(NULL, iniPath, MAX_PATH);
    char *slash = strrchr(iniPath, '\\');
    if (slash) strcpy(slash + 1, "lcse_hook.ini");
    else strcpy(iniPath, "lcse_hook.ini");
    GetPrivateProfileStringA("Font", "Name", "", g_fontName, LF_FACESIZE, iniPath);
    if (g_fontName[0] != '\0') g_fontOverride = TRUE;
    g_debugLog = GetPrivateProfileIntA("Debug", "Log", 0, iniPath);
}

BOOL WINAPI DllMain(HINSTANCE hinstDLL, DWORD fdwReason, LPVOID lpvReserved)
{
    if (fdwReason == DLL_PROCESS_ATTACH) {
        DisableThreadLibraryCalls(hinstDLL);
        LoadConfig();

        if (g_debugLog) {
            char logPath[MAX_PATH];
            GetModuleFileNameA(NULL, logPath, MAX_PATH);
            char *s = strrchr(logPath, '\\');
            if (s) strcpy(s + 1, "lcse_hook.log");
            else strcpy(logPath, "lcse_hook.log");
            g_logFile = fopen(logPath, "w");
            LogMsg("=== lcse_hook v5.2 ===\n");
            LogMsg("Font: '%s'\n", g_fontName);
            LogMsg("Accent bytes: 0xA1-0xAD (sign-ext: 0xFFFFFFA1-0xFFFFFFAD)\n");
        }

        char fontPath[MAX_PATH];
        GetModuleFileNameA(NULL, fontPath, MAX_PATH);
        char *s = strrchr(fontPath, '\\');
        if (s) strcpy(s + 1, "lcse_font.ttf");
        else strcpy(fontPath, "lcse_font.ttf");
        AddFontResourceExA(fontPath, FR_PRIVATE, 0);

        HMODULE hExe = GetModuleHandleA(NULL);
        PatchIAT(hExe, "GDI32.dll", "CreateFontIndirectA",
                 (void*)Hook_CreateFontIndirectA,
                 (void**)&g_origCreateFontIndirectA);
        PatchIAT(hExe, "GDI32.dll", "GetGlyphOutlineA",
                 (void*)Hook_GetGlyphOutlineA,
                 (void**)&g_origGetGlyphOutlineA);
    }
    else if (fdwReason == DLL_PROCESS_DETACH) {
        if (g_logFile) {
            LogMsg("=== unloaded ===\n");
            fclose(g_logFile);
            g_logFile = NULL;
        }
    }
    return TRUE;
}

__declspec(dllexport) void lcse_hook_init(void) {}
