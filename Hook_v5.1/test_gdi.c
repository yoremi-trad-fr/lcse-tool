/* Runtime regression test for the DLL only; never starts a game. */
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdio.h>
#include <string.h>
typedef DWORD (WINAPI *GlyphFn)(HDC,UINT,UINT,LPGLYPHMETRICS,DWORD,LPVOID,const MAT2*);
typedef HFONT (WINAPI *FontFn)(const LOGFONTA*);

int main(int argc,char **argv) {
    if(argc!=1&&argc!=2) return 2;
    if(argc==1) {
        if(!freopen("runtime-result.txt","w",stdout)) return 10;
        setvbuf(stdout,NULL,_IONBF,0);
        char cwd[MAX_PATH],exe[MAX_PATH];
        GetCurrentDirectoryA(MAX_PATH,cwd);GetModuleFileNameA(NULL,exe,MAX_PATH);
        char *slash=strrchr(exe,'\\');if(!slash)return 10;*slash=0;
        printf("ACP=%u LCID=0x%04lX cwd=%s\n",GetACP(),GetUserDefaultLCID(),cwd);
        if(GetACP()!=932||GetUserDefaultLCID()!=1041||_stricmp(cwd,exe))return 11;
    }
    HMODULE dll=LoadLibraryA(argc==1?"lcse_hook.dll":argv[1]);
    if(!dll){printf("LoadLibrary error %lu\n",GetLastError());return 3;}
    GlyphFn glyph=(GlyphFn)(void*)GetProcAddress(dll,"GetGlyphOutlineA");
    FontFn font=(FontFn)(void*)GetProcAddress(dll,"CreateFontIndirectA");
    if(!glyph||!font) return 4;
    if(argc==1&&glyph!=(GlyphFn)GetGlyphOutlineA) {
        puts("normal import did not resolve to accent DLL");return 12;
    }
    HMODULE system=GetModuleHandleA("GDI32.dll");
    const char *forwarded[]={"CreateDIBSection","DeleteObject","EnumFontFamiliesA",
        "GetClipBox","GetDCOrgEx","GetDeviceCaps","GetStockObject","SelectObject","SetDIBitsToDevice"};
    for(unsigned i=0;i<sizeof(forwarded)/sizeof(forwarded[0]);i++) {
        FARPROC expected=GetProcAddress(system,forwarded[i]);
        FARPROC got=GetProcAddress(dll,forwarded[i]);
        if(!got||got!=expected){printf("forward failed %s\n",forwarded[i]);return 9;}
    }
    LOGFONTA lf={0};lf.lfHeight=-24;lf.lfCharSet=SHIFTJIS_CHARSET;strcpy(lf.lfFaceName,"MS Gothic");
    HDC dc=CreateCompatibleDC(NULL);HFONT f=font(&lf);
    if(!dc||!f)return 5;
    HGDIOBJ previous=SelectObject(dc,f);
    const WCHAR unicode[13]={0xE9,0xE8,0xE7,0xE0,0xE2,0xFB,0xF4,0xEA,0xEE,0xF9,0xEB,0xEF,0xFC};
    MAT2 mat={{0,1},{0,0},{0,0},{0,1}};
    for(unsigned i=0;i<13;i++) {
        GLYPHMETRICS expected={0},actual={0};
        DWORD size=GetGlyphOutlineW(dc,unicode[i],GGO_GRAY8_BITMAP,&expected,0,NULL,&mat);
        if(size==GDI_ERROR)return 6;
        BYTE *a=(BYTE*)HeapAlloc(GetProcessHeap(),HEAP_ZERO_MEMORY,size?size:1);
        BYTE *b=(BYTE*)HeapAlloc(GetProcessHeap(),HEAP_ZERO_MEMORY,size?size:1);
        GetGlyphOutlineW(dc,unicode[i],GGO_GRAY8_BITMAP,&expected,size,a,&mat);
        for(unsigned sign=0;sign<2;sign++) {
            UINT byte=0xA1+i;if(sign)byte|=0xFFFFFF00;
            DWORD got=glyph(dc,byte,GGO_GRAY8_BITMAP,&actual,size,b,&mat);
            if(got!=size||memcmp(&expected,&actual,sizeof(expected))||memcmp(a,b,size)){printf("accent failed %u sign %u\n",i,sign);return 7;}
        }
        HeapFree(GetProcessHeap(),0,a);HeapFree(GetProcessHeap(),0,b);
    }
    for(unsigned i=0;i<2;i++) {
        UINT ch=i?0x82A0:'A';GLYPHMETRICS a={0},b={0};
        DWORD expected=GetGlyphOutlineA(dc,ch,GGO_METRICS,&a,0,NULL,&mat);
        DWORD got=glyph(dc,ch,GGO_METRICS,&b,0,NULL,&mat);
        if(got!=expected||memcmp(&a,&b,sizeof(a)))return 8;
    }
    SelectObject(dc,previous);DeleteObject(f);DeleteDC(dc);FreeLibrary(dll);
    puts("PASS: 13 accents, 26 byte/sign-extension forms, ASCII, Shift-JIS, 9 system GDI forwards");
    return 0;
}
