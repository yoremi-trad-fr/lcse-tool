/*
 * lcse_hook.dll - Font hook for LCSE engine French translation
 * 
 * Hooks CreateFontIndirectA ONLY to swap the font name.
 * The custom font (lcse_font.ttf) must have accent glyphs
 * at PUA positions U+E000-U+E01A (mapped from SJIS F040-F05A).
 *
 * Cross-compile: i686-w64-mingw32-gcc -shared -o lcse_hook.dll lcse_hook.c -lgdi32
 */
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdio.h>

typedef HFONT (WINAPI *pfnCreateFontIndirectA)(const LOGFONTA*);
static pfnCreateFontIndirectA g_origCreateFontIndirectA = NULL;

static char g_fontName[LF_FACESIZE] = "";
static BOOL g_fontOverride = FALSE;

static HFONT WINAPI Hook_CreateFontIndirectA(const LOGFONTA *lplf)
{
    if (g_fontOverride && lplf != NULL) {
        LOGFONTA lf = *lplf;
        strncpy(lf.lfFaceName, g_fontName, LF_FACESIZE - 1);
        lf.lfFaceName[LF_FACESIZE - 1] = '\0';
        return g_origCreateFontIndirectA(&lf);
    }
    return g_origCreateFontIndirectA(lplf);
}

static BOOL PatchIAT(HMODULE hModule, const char *dllName,
                     const char *funcName, void *hookFunc, void **origFunc)
{
    PIMAGE_DOS_HEADER dosHeader = (PIMAGE_DOS_HEADER)hModule;
    PIMAGE_NT_HEADERS ntHeaders = (PIMAGE_NT_HEADERS)((BYTE*)hModule + dosHeader->e_lfanew);
    PIMAGE_IMPORT_DESCRIPTOR imports = (PIMAGE_IMPORT_DESCRIPTOR)(
        (BYTE*)hModule + ntHeaders->OptionalHeader.DataDirectory[IMAGE_DIRECTORY_ENTRY_IMPORT].VirtualAddress);

    for (; imports->Name; imports++) {
        const char *name = (const char*)((BYTE*)hModule + imports->Name);
        if (_stricmp(name, dllName) != 0) continue;
        PIMAGE_THUNK_DATA origThunk = (PIMAGE_THUNK_DATA)((BYTE*)hModule + imports->OriginalFirstThunk);
        PIMAGE_THUNK_DATA firstThunk = (PIMAGE_THUNK_DATA)((BYTE*)hModule + imports->FirstThunk);
        for (; origThunk->u1.AddressOfData; origThunk++, firstThunk++) {
            if (origThunk->u1.Ordinal & IMAGE_ORDINAL_FLAG) continue;
            PIMAGE_IMPORT_BY_NAME importByName = (PIMAGE_IMPORT_BY_NAME)(
                (BYTE*)hModule + origThunk->u1.AddressOfData);
            if (strcmp((const char*)importByName->Name, funcName) == 0) {
                DWORD oldProtect;
                VirtualProtect(&firstThunk->u1.Function, sizeof(void*), PAGE_READWRITE, &oldProtect);
                *origFunc = (void*)firstThunk->u1.Function;
                firstThunk->u1.Function = (ULONG_PTR)hookFunc;
                VirtualProtect(&firstThunk->u1.Function, sizeof(void*), oldProtect, &oldProtect);
                return TRUE;
            }
        }
    }
    return FALSE;
}

static void LoadConfig(void)
{
    char iniPath[MAX_PATH];
    GetModuleFileNameA(NULL, iniPath, MAX_PATH);
    char *slash = strrchr(iniPath, '\\');
    if (slash) strcpy(slash + 1, "lcse_hook.ini");
    else strcpy(iniPath, "lcse_hook.ini");
    GetPrivateProfileStringA("Font", "Name", "", g_fontName, LF_FACESIZE, iniPath);
    if (g_fontName[0] != '\0') g_fontOverride = TRUE;
}

BOOL WINAPI DllMain(HINSTANCE hinstDLL, DWORD fdwReason, LPVOID lpvReserved)
{
    if (fdwReason == DLL_PROCESS_ATTACH) {
        DisableThreadLibraryCalls(hinstDLL);
        LoadConfig();
        HMODULE hExe = GetModuleHandleA(NULL);
        PatchIAT(hExe, "GDI32.dll", "CreateFontIndirectA",
                 (void*)Hook_CreateFontIndirectA, (void**)&g_origCreateFontIndirectA);
        char fontPath[MAX_PATH];
        GetModuleFileNameA(NULL, fontPath, MAX_PATH);
        char *s = strrchr(fontPath, '\\');
        if (s) strcpy(s + 1, "lcse_font.ttf");
        else strcpy(fontPath, "lcse_font.ttf");
        AddFontResourceExA(fontPath, FR_PRIVATE, 0);
    }
    return TRUE;
}

__declspec(dllexport) void lcse_hook_init(void) {}
