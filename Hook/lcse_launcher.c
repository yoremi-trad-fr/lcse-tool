/*
 * lcse_launcher.exe - Launches lcsebody.exe with lcse_hook.dll injected
 *
 * Cross-compile: i686-w64-mingw32-gcc -o lcse_launcher.exe lcse_launcher.c
 */
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdio.h>

int main(void)
{
    /* Get our directory */
    char dir[MAX_PATH], exePath[MAX_PATH], dllPath[MAX_PATH];
    GetModuleFileNameA(NULL, dir, MAX_PATH);
    char *slash = strrchr(dir, '\\');
    if (slash) *(slash + 1) = '\0'; else dir[0] = '\0';

    snprintf(exePath, MAX_PATH, "%slcsebody.exe", dir);
    snprintf(dllPath, MAX_PATH, "%slcse_hook.dll", dir);

    /* Check files exist */
    if (GetFileAttributesA(exePath) == INVALID_FILE_ATTRIBUTES) {
        MessageBoxA(NULL, "lcsebody.exe not found!", "LCSE Launcher", MB_ICONERROR);
        return 1;
    }
    if (GetFileAttributesA(dllPath) == INVALID_FILE_ATTRIBUTES) {
        MessageBoxA(NULL, "lcse_hook.dll not found!", "LCSE Launcher", MB_ICONERROR);
        return 1;
    }

    /* Create process suspended */
    STARTUPINFOA si = { .cb = sizeof(si) };
    PROCESS_INFORMATION pi = {0};
    if (!CreateProcessA(exePath, NULL, NULL, NULL, FALSE,
                        CREATE_SUSPENDED, NULL, dir, &si, &pi)) {
        MessageBoxA(NULL, "Failed to start lcsebody.exe", "LCSE Launcher", MB_ICONERROR);
        return 1;
    }

    /* Allocate memory in target process for DLL path */
    size_t pathLen = strlen(dllPath) + 1;
    LPVOID remoteMem = VirtualAllocEx(pi.hProcess, NULL, pathLen,
                                       MEM_COMMIT | MEM_RESERVE, PAGE_READWRITE);
    if (!remoteMem) {
        MessageBoxA(NULL, "VirtualAllocEx failed", "LCSE Launcher", MB_ICONERROR);
        TerminateProcess(pi.hProcess, 1);
        return 1;
    }

    /* Write DLL path to target process */
    WriteProcessMemory(pi.hProcess, remoteMem, dllPath, pathLen, NULL);

    /* Get LoadLibraryA address (same in all processes) */
    HMODULE hKernel32 = GetModuleHandleA("kernel32.dll");
    FARPROC pLoadLibrary = GetProcAddress(hKernel32, "LoadLibraryA");

    /* Create remote thread to call LoadLibraryA(dllPath) */
    HANDLE hThread = CreateRemoteThread(pi.hProcess, NULL, 0,
        (LPTHREAD_START_ROUTINE)pLoadLibrary, remoteMem, 0, NULL);
    if (!hThread) {
        MessageBoxA(NULL, "CreateRemoteThread failed", "LCSE Launcher", MB_ICONERROR);
        TerminateProcess(pi.hProcess, 1);
        return 1;
    }

    /* Wait for DLL to load */
    WaitForSingleObject(hThread, INFINITE);
    CloseHandle(hThread);
    VirtualFreeEx(pi.hProcess, remoteMem, 0, MEM_RELEASE);

    /* Resume the main thread */
    ResumeThread(pi.hThread);

    CloseHandle(pi.hThread);
    CloseHandle(pi.hProcess);
    return 0;
}
