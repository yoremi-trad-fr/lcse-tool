package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

const (fnSz = 0x40; lstESz = fnSz + 12; instrSz = 12)

var tEM = map[uint32]string{1:"snx",2:"bmp",3:"png",4:"wav",5:"ogg"}
var eTM = map[string]uint32{"snx":1,"bmp":2,"png":3,"wav":4,"ogg":5}

func s2u(b []byte)(string,error){r:=transform.NewReader(bytes.NewReader(b),japanese.ShiftJIS.NewDecoder());o,e:=io.ReadAll(r);return string(o),e}
func u2s(s string)([]byte,error){r:=transform.NewReader(strings.NewReader(s),japanese.ShiftJIS.NewEncoder());return io.ReadAll(r)}
func exK(b byte)uint32{k:=uint32(b);return k|(k<<8)|(k<<16)|(k<<24)}
func xB(d []byte,k byte)[]byte{o:=make([]byte,len(d));for i,b:=range d{o[i]=b^k};return o}
func adK(p string)(byte,error){f,e:=os.Open(p);if e!=nil{return 0,e};defer f.Close();fi,_:=f.Stat();var b[4]byte;f.Read(b[:]);k:=b[3];if k==0{return 0,fmt.Errorf("no key")};if int(binary.LittleEndian.Uint32(b[:])^exK(k))*lstESz+4==int(fi.Size()){return k,nil};return 0,fmt.Errorf("fail")}

type LE struct{O,S uint32;N string;T uint32;E string}
func pLST(p string,k byte)([]LE,error){d,e:=os.ReadFile(p);if e!=nil{return nil,e};xk:=exK(k);c:=binary.LittleEndian.Uint32(d[0:4])^xk;if int(c)*lstESz+4!=len(d){return nil,fmt.Errorf("mismatch")};es:=make([]LE,c);pos:=4;for i:=uint32(0);i<c;i++{o:=binary.LittleEndian.Uint32(d[pos:pos+4])^xk;s:=binary.LittleEndian.Uint32(d[pos+4:pos+8])^xk;pos+=8;nb:=make([]byte,fnSz);copy(nb,d[pos:pos+fnSz]);pos+=fnSz;nl:=0;for j:=0;j<fnSz;j++{if nb[j]==0{break};if nb[j]!=k{nb[j]^=k};nl=j+1};n,_:=s2u(nb[:nl]);t:=binary.LittleEndian.Uint32(d[pos:pos+4]);pos+=4;ext:=tEM[t];if ext==""{ext=fmt.Sprintf("%d",t)};es[i]=LE{o,s,n,t,ext}};return es,nil}
func rK(p string,ko,so int)(byte,byte,error){var ik,sk byte;if ko>=0{ik=byte(ko)}else{d,e:=adK(p);if e!=nil{return 0,0,e};ik=d};if so>=0{sk=byte(so)}else{sk=ik+1};return ik,sk,nil}

func cmdUnpack(pp,od string,ko,so int)error{ik,sk,e:=rK(pp+".lst",ko,so);if e!=nil{return e};es,e:=pLST(pp+".lst",ik);if e!=nil{return e};pf,_:=os.Open(pp);defer pf.Close();os.MkdirAll(od,0755);for _,e:=range es{fn:=e.N+"."+e.E;d:=make([]byte,e.S);pf.ReadAt(d,int64(e.O));if e.E=="snx"{d=xB(d,sk)};os.WriteFile(filepath.Join(od,fn),d,0644)};fmt.Printf("[INFO] %d files -> %s\n",len(es),od);return nil}
func cmdPatch(op,pd,out string,ko,so int)error{ik,sk,e:=rK(op+".lst",ko,so);if e!=nil{return e};es,e:=pLST(op+".lst",ik);if e!=nil{return e};pf,_:=os.Open(op);defer pf.Close();rp:=map[string]string{};filepath.Walk(pd,func(p string,i os.FileInfo,e error)error{if e!=nil||i.IsDir(){return e};rp[strings.ToUpper(filepath.Base(p))]=p;return nil});of,_:=os.Create(out);defer of.Close();lf,_:=os.Create(out+".lst");defer lf.Close();xk:=exK(ik);var b[4]byte;binary.LittleEndian.PutUint32(b[:],uint32(len(es))^xk);lf.Write(b[:]);pc:=0;for _,e:=range es{fn:=e.N+"."+e.E;var d[]byte;if r,ok:=rp[strings.ToUpper(fn)];ok{d,_=os.ReadFile(r);if e.E=="snx"{d=xB(d,sk)};fmt.Printf("[PATCH] %s\n",fn);pc++}else{d=make([]byte,e.S);pf.ReadAt(d,int64(e.O))};po,_:=of.Seek(0,io.SeekCurrent);binary.LittleEndian.PutUint32(b[:],uint32(po)^xk);lf.Write(b[:]);binary.LittleEndian.PutUint32(b[:],uint32(len(d))^xk);lf.Write(b[:]);sn,_:=u2s(e.N);if sn==nil{sn=[]byte(e.N)};nb:=make([]byte,fnSz);for i,v:=range sn{if i<fnSz{nb[i]=v^ik}};lf.Write(nb);binary.LittleEndian.PutUint32(b[:],e.T);lf.Write(b[:]);of.Write(d)};fmt.Printf("[INFO] %d/%d patched -> %s\n",pc,len(es),out);return nil}
func cmdPack(sd,out string,ko,so int)error{ik:=byte(0x01);sk:=byte(0x02);if ko>=0{ik=byte(ko);sk=ik+1};if so>=0{sk=byte(so)};var fs[]string;filepath.Walk(sd,func(p string,i os.FileInfo,e error)error{if e!=nil||i.IsDir(){return e};fs=append(fs,p);return nil});sort.Strings(fs);pf,_:=os.Create(out);defer pf.Close();lf,_:=os.Create(out+".lst");defer lf.Close();xk:=exK(ik);var b[4]byte;binary.LittleEndian.PutUint32(b[:],uint32(len(fs))^xk);lf.Write(b[:]);for _,fp:=range fs{d,_:=os.ReadFile(fp);ext:=strings.ToLower(strings.TrimPrefix(filepath.Ext(fp),"."));bn:=strings.TrimSuffix(filepath.Base(fp),filepath.Ext(fp));po,_:=pf.Seek(0,io.SeekCurrent);binary.LittleEndian.PutUint32(b[:],uint32(po)^xk);lf.Write(b[:]);binary.LittleEndian.PutUint32(b[:],uint32(len(d))^xk);lf.Write(b[:]);sn,_:=u2s(bn);if sn==nil{sn=[]byte(bn)};nb:=make([]byte,fnSz);for i,v:=range sn{nb[i]=v^ik};lf.Write(nb);binary.LittleEndian.PutUint32(b[:],eTM[ext]);lf.Write(b[:]);if ext=="snx"{d=xB(d,sk)};pf.Write(d)};return nil}

// ─── SNX ─────────────────────────────────────────────────────────────────────

type SE struct{ Off, DLen uint32; Raw []byte }

func pSNX(d []byte) (h0, h1 uint32, bc []byte, entries []SE, err error) {
	if len(d) < 8 { return 0,0,nil,nil,fmt.Errorf("too small") }
	h0 = binary.LittleEndian.Uint32(d[0:4])
	h1 = binary.LittleEndian.Uint32(d[4:8])
	if int(h1) > len(d) { return 0,0,nil,nil,fmt.Errorf("overflow") }
	ss := uint32(len(d)) - h1
	bc = make([]byte, ss); copy(bc, d[:ss])
	pos := uint32(0)
	for pos+4 <= h1 {
		sl := binary.LittleEndian.Uint32(d[ss+pos : ss+pos+4])
		if sl == 0 || pos+4+sl > h1 { break }
		r := make([]byte, sl); copy(r, d[ss+pos+4:ss+pos+4+sl])
		entries = append(entries, SE{pos, sl, r}); pos += 4 + sl
	}
	return
}

func cTxt(r []byte) ([]byte, bool) {
	d := r; if len(d)>0 && d[len(d)-1]==0 { d=d[:len(d)-1] }
	if len(d)>=2 && d[len(d)-2]==0x02 && d[len(d)-1]==0x03 { return d[:len(d)-2], true }
	return d, false
}
func cjk(s string) bool {
	for _, r := range s { if (r>=0x3000&&r<=0x9FFF)||(r>=0xF900&&r<=0xFAFF) { return true } }
	return false
}

func cmdSNX2TXT(sp, op string) error {
	d, _ := os.ReadFile(sp)
	_, _, _, entries, err := pSNX(d); if err != nil { return err }
	if op == "" { op = strings.TrimSuffix(sp, filepath.Ext(sp)) + ".txt" }
	f, _ := os.Create(op); defer f.Close()
	w := bufio.NewWriter(f); defer w.Flush()
	w.WriteString("# LCSE SNX: " + filepath.Base(sp) + "\r\n")
	w.WriteString("# INDEX\\tOFFSET\\tTYPE\\tTEXT (Shift-JIS)\r\n#\r\n")
	dl := 0
	for i, e := range entries {
		cl, _ := cTxt(e.Raw); txt, _ := s2u(cl)
		t := "RES"; if cjk(txt) { t="TXT"; dl++ } else {
			ok:=false; for _,b:=range cl{if b>=0x20&&b<=0x7E{ok=true}}
			if !ok&&len(cl)<=2{t="CTL"}
		}
		fmt.Fprintf(w, "%d\t0x%04X\t%s\t", i, e.Off, t); w.Write(cl); w.WriteString("\r\n")
	}
	fmt.Printf("[INFO] %s: %d strings (%d dialogue) -> %s\n", filepath.Base(sp), len(entries), dl, op)
	return nil
}

// ─── txt2snx — FULL REBUILD + 12-byte aligned ref update ────────────────────
// Instructions are 12 bytes: [opcode][arg1][arg2]
// String refs: opcode=0x11, arg1=0x02, arg2=string_table_offset

func cmdTXT2SNX(tp, sp, op string) error {
	od, _ := os.ReadFile(sp)
	h0, _, bc, entries, err := pSNX(od); if err != nil { return err }
	td, _ := os.ReadFile(tp)
	if op == "" { op = strings.TrimSuffix(sp, filepath.Ext(sp)) + "_patched" + filepath.Ext(sp) }

	// Parse SJIS text (4 cols: idx, off, type, text)
	tm := map[int][]byte{}
	for _, line := range bytes.Split(td, []byte("\n")) {
		line = bytes.TrimRight(line, "\r")
		if len(line) == 0 || line[0] == '#' { continue }
		parts := bytes.SplitN(line, []byte("\t"), 4)
		if len(parts) < 4 { continue }
		var idx int
		if _, e := fmt.Sscanf(string(parts[0]), "%d", &idx); e != nil { continue }
		if len(parts[3]) > 0 { tm[idx] = append([]byte{}, parts[3]...) }
	}

	// Validate 12-byte alignment: 8 + h0*12 + h1 must equal file size
	expectedSize := 8 + h0*instrSz + binary.LittleEndian.Uint32(od[4:8])
	if expectedSize != uint32(len(od)) {
		// Non-standard SNX (extra data between bytecode and strings) — copy as-is
		os.WriteFile(op, od, 0644)
		fmt.Printf("[SKIP] %s: non-standard format (expected %d, got %d) -> copie intacte\n",
			filepath.Base(sp), expectedSize, len(od))
		return nil
	}

	// PHASE 1: Count changes first (dry run)
	changed := 0
	for i, e := range entries {
		if newTxt, ok := tm[i]; ok {
			_, hc := cTxt(e.Raw)
			var nd []byte
			nd = append(nd, newTxt...)
			if hc { nd = append(nd, 0x02, 0x03) }
			nd = append(nd, 0x00)
			if !bytes.Equal(nd, e.Raw) { changed++ }
		}
	}

	// If ZERO changes, copy original byte-for-byte (SAFE)
	if changed == 0 {
		os.WriteFile(op, od, 0644)
		origH1 := binary.LittleEndian.Uint32(od[4:8])
		fmt.Printf("[INFO] 0 modifiees -> %s (copie intacte, %d bytes)\n", op, len(od))
		_ = origH1
		return nil
	}

	// PHASE 2: Rebuild string table — all entries packed sequentially
	oldToNew := make(map[uint32]uint32)
	var newTable bytes.Buffer

	for i, e := range entries {
		newOff := uint32(newTable.Len())
		oldToNew[e.Off] = newOff

		var entryData []byte
		if newTxt, ok := tm[i]; ok {
			_, hc := cTxt(e.Raw)
			entryData = append(entryData, newTxt...)
			if hc { entryData = append(entryData, 0x02, 0x03) }
			entryData = append(entryData, 0x00)
		} else {
			entryData = e.Raw
		}

		var lb [4]byte
		binary.LittleEndian.PutUint32(lb[:], uint32(len(entryData)))
		newTable.Write(lb[:])
		newTable.Write(entryData)
	}

	// PHASE 3: Update refs at 12-byte instruction boundaries
	newBC := make([]byte, len(bc)); copy(newBC, bc)
	refsUpdated := 0
	for i := uint32(0); i < h0; i++ {
		off := 8 + i*instrSz
		opcode := binary.LittleEndian.Uint32(newBC[off : off+4])
		arg1   := binary.LittleEndian.Uint32(newBC[off+4 : off+8])
		arg2   := binary.LittleEndian.Uint32(newBC[off+8 : off+12])
		if opcode == 0x11 && arg1 == 0x02 {
			if newOff, ok := oldToNew[arg2]; ok && newOff != arg2 {
				binary.LittleEndian.PutUint32(newBC[off+8:off+12], newOff)
				refsUpdated++
			}
		}
	}

	// Update h1 in header
	binary.LittleEndian.PutUint32(newBC[4:8], uint32(newTable.Len()))

	var out bytes.Buffer
	out.Write(newBC); out.Write(newTable.Bytes())
	os.WriteFile(op, out.Bytes(), 0644)

	newH1 := uint32(newTable.Len())
	origH1 := binary.LittleEndian.Uint32(od[4:8])
	fmt.Printf("[INFO] %d modifiees, %d refs maj, table: %d -> %d (%+d) -> %s\n",
		changed, refsUpdated, origH1, newH1, int(newH1)-int(origH1), op)
	return nil
}

func cmdSNX2TXTBatch(d, od string) error {
	if od=="" { od=d+"_txt" }; os.MkdirAll(od, 0755)
	m,_:=filepath.Glob(filepath.Join(d,"*.[sS][nN][xX]"))
	if len(m)==0{return fmt.Errorf("no SNX")}
	for _,f:=range m{b:=strings.TrimSuffix(filepath.Base(f),filepath.Ext(f));cmdSNX2TXT(f,filepath.Join(od,b+".txt"))}
	return nil
}
func cmdTXT2SNXBatch(td, sd, od string) error {
	if od=="" { od=sd+"_patched" }; os.MkdirAll(od, 0755)
	m,_:=filepath.Glob(filepath.Join(td,"*.txt"))
	if len(m)==0{return fmt.Errorf("no TXT")}
	for _,f:=range m{b:=strings.TrimSuffix(filepath.Base(f),filepath.Ext(f));sp:=filepath.Join(sd,b+".SNX");if _,e:=os.Stat(sp);e!=nil{sp=filepath.Join(sd,b+".snx");if _,e:=os.Stat(sp);e!=nil{continue}};cmdTXT2SNX(f,sp,filepath.Join(od,b+".snx"))}
	return nil
}

func usage(){fmt.Fprintf(os.Stderr,`lcse-tool v0.6.1 - LC-ScriptEngine

PLUS DE LIMITE DE TAILLE sur les traductions !
La table de chaines est reconstruite et les references mises a jour
par scan 12-octets aligne (format d'instruction LCSE confirme).

ARCHIVE:
  lcse-tool unpack <lcsebody> [output_dir]
  lcse-tool patch <original> <patches_dir> <sortie>
  lcse-tool pack <dir> <sortie>

SCRIPTS:
  lcse-tool snx2txt <file.snx|dir> [output]
  lcse-tool txt2snx <text.txt> <original.snx> [output.snx]
  lcse-tool txt2snx-batch <txt_dir> <snx_dir> [output_dir]

OPTIONS:  --key <hex>  --snxkey <hex>
`)}

func main(){
	args:=os.Args[1:];ko,so:=-1,-1;var pos[]string
	for i:=0;i<len(args);i++{switch args[i]{
	case "--key":if i+1<len(args){var k int;fmt.Sscanf(args[i+1],"%x",&k);ko=k;i++}
	case "--snxkey":if i+1<len(args){var k int;fmt.Sscanf(args[i+1],"%x",&k);so=k;i++}
	case "-h","--help":usage();os.Exit(0)
	default:pos=append(pos,args[i])}}
	if len(pos)<1{usage();os.Exit(1)}
	cmd,ca:=pos[0],pos[1:];var err error
	switch cmd{
	case "unpack","u":if len(ca)<1{usage();os.Exit(1)};od:=ca[0]+"_extracted";if len(ca)>=2{od=ca[1]};err=cmdUnpack(ca[0],od,ko,so)
	case "patch":if len(ca)<3{usage();os.Exit(1)};err=cmdPatch(ca[0],ca[1],ca[2],ko,so)
	case "pack","p":if len(ca)<2{usage();os.Exit(1)};err=cmdPack(ca[0],ca[1],ko,so)
	case "snx2txt","s2t":if len(ca)<1{usage();os.Exit(1)};o:="";if len(ca)>=2{o=ca[1]};if inf,_:=os.Stat(ca[0]);inf!=nil&&inf.IsDir(){err=cmdSNX2TXTBatch(ca[0],o)}else{err=cmdSNX2TXT(ca[0],o)}
	case "txt2snx","t2s":if len(ca)<2{usage();os.Exit(1)};o:="";if len(ca)>=3{o=ca[2]};err=cmdTXT2SNX(ca[0],ca[1],o)
	case "txt2snx-batch","t2s-batch":if len(ca)<2{usage();os.Exit(1)};o:="";if len(ca)>=3{o=ca[2]};err=cmdTXT2SNXBatch(ca[0],ca[1],o)
	default:fmt.Fprintf(os.Stderr,"Unknown: %s\n",cmd);os.Exit(1)}
	if err!=nil{fmt.Fprintf(os.Stderr,"[ERROR] %v\n",err);os.Exit(1)}
}
