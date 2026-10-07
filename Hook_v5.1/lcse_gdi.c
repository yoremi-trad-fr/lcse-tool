/* LCSE accents 1.4: normal PE import, no process injection or IAT rewriting.
 * Only the installer-created copy of the game imports this DLL.
 * Other GDI exports are forwarded to the system GDI32 by lcse_gdi.def.
 * Build: i686-w64-mingw32-gcc -shared -O2 -Wall -Wextra -o lcse_hook.dll
 *        lcse_gdi.c lcse_gdi.def -lgdi32 -Wl,--kill-at
 */
#define WIN32_LEAN_AND_MEAN
#define _WIN32_WINNT 0x0600
#include <windows.h>
#include <stdio.h>
#include <stdarg.h>

static INIT_ONCE configOnce = INIT_ONCE_STATIC_INIT;
static char fontName[LF_FACESIZE];
static FILE *logFile;
static SRWLOCK logLock = SRWLOCK_INIT;
static const WCHAR accents[13] = {
    0xE9, 0xE8, 0xE7, 0xE0, 0xE2, 0xFB, 0xF4,
    0xEA, 0xEE, 0xF9, 0xEB, 0xEF, 0xFC
};

static BOOL siblingPath(WCHAR *path, const WCHAR *name) {
    DWORD length = GetModuleFileNameW(NULL, path, MAX_PATH);
    if (!length || length >= MAX_PATH) return FALSE;
    WCHAR *slash = wcsrchr(path, L'\\');
    if (!slash || (size_t)(slash - path + 1) + wcslen(name) >= MAX_PATH) return FALSE;
    wcscpy(slash + 1, name);
    return TRUE;
}

static BOOL CALLBACK loadConfig(PINIT_ONCE once, PVOID param, PVOID *ctx) {
    (void)once; (void)param; (void)ctx;
    WCHAR path[MAX_PATH], name[LF_FACESIZE];
    if (siblingPath(path, L"lcse_hook.ini")) {
        GetPrivateProfileStringW(L"Font", L"Name", L"MS Gothic", name, LF_FACESIZE, path);
        WideCharToMultiByte(CP_ACP, 0, name, -1, fontName, LF_FACESIZE, NULL, NULL);
        fontName[LF_FACESIZE - 1] = '\0';
        if (GetPrivateProfileIntW(L"Debug", L"Log", 0, path) && siblingPath(path, L"lcse_hook.log")) {
            logFile = _wfopen(path, L"w");
            if (logFile) { fputs("LCSE accents 1.4 - normal GDI import\n", logFile); fflush(logFile); }
        }
    }
    if (siblingPath(path, L"lcse_font.ttf") && GetFileAttributesW(path) != INVALID_FILE_ATTRIBUTES)
        AddFontResourceExW(path, FR_PRIVATE, NULL);
    return TRUE;
}

static void logMessage(const char *format, ...) {
    if (!logFile) return;
    AcquireSRWLockExclusive(&logLock);
    va_list args;
    va_start(args, format);
    vfprintf(logFile, format, args);
    va_end(args);
    fflush(logFile);
    ReleaseSRWLockExclusive(&logLock);
}

HFONT WINAPI LCSE_CreateFontIndirectA(const LOGFONTA *original) {
    InitOnceExecuteOnce(&configOnce, loadConfig, NULL, NULL);
    if (original && fontName[0]) {
        LOGFONTA font = *original;
        lstrcpynA(font.lfFaceName, fontName, LF_FACESIZE);
        if (font.lfCharSet == DEFAULT_CHARSET || font.lfCharSet == SHIFTJIS_CHARSET)
            font.lfCharSet = SHIFTJIS_CHARSET;
        logMessage("[FONT] %s\n", font.lfFaceName);
        return CreateFontIndirectA(&font);
    }
    return CreateFontIndirectA(original);
}

DWORD WINAPI LCSE_GetGlyphOutlineA(HDC dc, UINT ch, UINT format,
        LPGLYPHMETRICS metrics, DWORD size, LPVOID buffer, const MAT2 *matrix) {
    InitOnceExecuteOnce(&configOnce, loadConfig, NULL, NULL);
    UINT byte = ch & 0xFF;
    if (((ch & 0xFFFFFF00U) == 0xFFFFFF00U || ch <= 0xFF) && byte >= 0xA1 && byte <= 0xAD) {
        DWORD result = GetGlyphOutlineW(dc, accents[byte - 0xA1], format, metrics, size, buffer, matrix);
        logMessage("[ACCENT] 0x%08X -> U+%04X\n", ch, (unsigned)accents[byte - 0xA1]);
        return result;
    }
    return GetGlyphOutlineA(dc, ch, format, metrics, size, buffer, matrix);
}

BOOL WINAPI DllMain(HINSTANCE instance, DWORD reason, LPVOID reserved) {
    (void)instance; (void)reason; (void)reserved;
    /* Configuration and font registration run on the first GDI call,
       outside the loader lock. Process teardown closes the log handle. */
    return TRUE;
}
