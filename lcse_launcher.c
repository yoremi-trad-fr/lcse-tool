/* LCSE 1.4: ordinary ONE launcher for an installer-created accented copy.
 * Build with -municode -mwindows. MOON uses MOON_FR.bat and Locale Emulator.
 * The game imports the DLL normally, with its original basename as argv[0].
 */
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <wchar.h>
int WINAPI wWinMain(HINSTANCE instance, HINSTANCE previous, LPWSTR args, int show) {
    (void)instance; (void)previous; (void)args; (void)show;
    WCHAR path[MAX_PATH], dir[MAX_PATH], command[MAX_PATH + 3];
    DWORD length = GetModuleFileNameW(NULL, dir, MAX_PATH);
    if (!length || length >= MAX_PATH) return 1;
    WCHAR *slash = wcsrchr(dir, L'\\');
    if (!slash) return 1;
    slash[1] = 0;
    const WCHAR *target = L"lcsebody.exe";
    if (wcslen(dir) + 8 + wcslen(target) >= MAX_PATH) return 1;
    wcscpy(path, dir); wcscat(path, L"lcse_fr\\"); wcscat(path, target);
    wcscpy(command, L"\""); wcscat(command, target); wcscat(command, L"\"");
    STARTUPINFOW startup = {0};
    PROCESS_INFORMATION process = {0};
    startup.cb = sizeof(startup);
    if (!CreateProcessW(path, command, NULL, NULL, FALSE, 0, NULL, dir, &startup, &process)) {
        MessageBoxW(NULL, L"Installez les accents avec LCSE Tool GUI 1.4 dans ce dossier.",
                    L"LCSE accents", MB_OK | MB_ICONERROR);
        return 1;
    }
    CloseHandle(process.hThread); CloseHandle(process.hProcess);
    return 0;
}
