/*
 * lcse_launcher.exe - Launches lcsebody.exe with lcse_hook.dll injected
 * Can be renamed freely (e.g. ONE_launcher.exe)
 */
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdio.h>

int WINAPI WinMain(HINSTANCE hInstance, HINSTANCE hPrevInstance,
                   LPSTR lpCmdLine, int nCmdShow)
{
    char myPath[MAX_PATH], dir[MAX_PATH], exePath[MAX_PATH], dllPath[MAX_PATH];
    
    /* Get OUR full path (works regardless of working directory) */
    GetModuleFileNameA(hInstance, myPath, MAX_PATH);
    
    /* Extract directory */
    lstrcpyA(dir, myPath);
    char *slash = strrchr(dir, '\\');
    if (slash) *(slash + 1) = '\0';
    else { dir[0] = '.'; dir[1] = '\\'; dir[2] = '\0'; }

    /* Set working directory to our folder (critical!) */
    SetCurrentDirectoryA(dir);

    /* Build paths */
    wsprintfA(exePath, "%slcsebody.exe", dir);
    wsprintfA(dllPath, "%slcse_hook.dll", dir);

    /* Check files exist */
    if (GetFileAttributesA(exePath) == INVALID_FILE_ATTRIBUTES) {
        char msg[MAX_PATH + 64];
        wsprintfA(msg, "lcsebody.exe not found!\n\nSearched in:\n%s", dir);
        MessageBoxA(NULL, msg, "LCSE Launcher", MB_ICONERROR);
        return 1;
    }
    if (GetFileAttributesA(dllPath) == INVALID_FILE_ATTRIBUTES) {
        MessageBoxA(NULL, "lcse_hook.dll not found!", "LCSE Launcher", MB_ICONERROR);
        return 1;
    }

    /* Create process suspended */
    STARTUPINFOA si;
    PROCESS_INFORMATION pi;
    ZeroMemory(&si, sizeof(si));
    si.cb = sizeof(si);
    ZeroMemory(&pi, sizeof(pi));
    
    if (!CreateProcessA(exePath, NULL, NULL, NULL, FALSE,
                        CREATE_SUSPENDED, NULL, dir, &si, &pi)) {
        MessageBoxA(NULL, "Failed to start lcsebody.exe", "LCSE Launcher", MB_ICONERROR);
        return 1;
    }

    /* Allocate memory in target for DLL path */
    size_t pathLen = lstrlenA(dllPath) + 1;
    LPVOID remoteMem = VirtualAllocEx(pi.hProcess, NULL, pathLen,
                                       MEM_COMMIT | MEM_RESERVE, PAGE_READWRITE);
    if (!remoteMem) {
        TerminateProcess(pi.hProcess, 1);
        MessageBoxA(NULL, "VirtualAllocEx failed", "LCSE Launcher", MB_ICONERROR);
        return 1;
    }

    WriteProcessMemory(pi.hProcess, remoteMem, dllPath, pathLen, NULL);

    /* Inject DLL */
    HMODULE hKernel32 = GetModuleHandleA("kernel32.dll");
    FARPROC pLoadLibrary = GetProcAddress(hKernel32, "LoadLibraryA");

    HANDLE hThread = CreateRemoteThread(pi.hProcess, NULL, 0,
        (LPTHREAD_START_ROUTINE)pLoadLibrary, remoteMem, 0, NULL);
    if (!hThread) {
        TerminateProcess(pi.hProcess, 1);
        MessageBoxA(NULL, "DLL injection failed", "LCSE Launcher", MB_ICONERROR);
        return 1;
    }

    WaitForSingleObject(hThread, INFINITE);
    CloseHandle(hThread);
    VirtualFreeEx(pi.hProcess, remoteMem, 0, MEM_RELEASE);

    /* Resume game */
    ResumeThread(pi.hThread);
    CloseHandle(pi.hThread);
    CloseHandle(pi.hProcess);
    return 0;
}
