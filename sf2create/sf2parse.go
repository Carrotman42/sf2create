
package sf2create

import (
    "fmt"
    "io"
)

func getChunkHeader(in io.Reader) (sFChild, io.Reader) {
    var header [12]byte;
    if n, _ := io.ReadFull(in, header[:]); n != 12 {
        return nil, nil
    }
    llen := toint64(header[4:8]) - 4 //Offset the 4 that we already read
    fmt.Println("got chunk len ", llen)
    limited := io.LimitReader(in, llen)
    ttype := [4]byte{header[8],header[9],header[10],header[11]}

    if header[0] == 'R' && header[1] == 'I' && header[2] == 'F' && header[3] == 'F' {
        ch,ok := riffChildren[ttype];
        if !ok {
            panic ("Invalid SoundFont file!")
        }
        return ch, limited
    }else if header[0] == 'L' && header[1] == 'I' && header[2] == 'S'&& header[3] == 'T'{
        ch,ok := listChildren[ttype]
        if !ok {
            panic (fmt.Sprint("Invalid LIST type: ", string(header[8:12])))
        }
        return ch, limited
    }

    panic(fmt.Sprint("Invalid header type: '", string(header[:4]), "'"))
}

func toint64(dword []byte) int64 {
    return int64((int64(dword[3]) << 24) | (int64(dword[2]) << 16) | (int64(dword[1]) << 8) |int64(dword[0]))
}

func Dump(in io.Reader) {
    child, newReader := getChunkHeader(in);
    child.dump("", newReader)
}

func skip(in io.Reader, llen int64) {
    var skipbuf [1024*4]byte;
    for llen > 0 {
        var sl []byte;
        if llen >= 1024*4 {
            sl = skipbuf[:]
        }else{
            sl = skipbuf[:int(llen)]
        }

        if n, err := in.Read(sl); n == 0 {
            fmt.Println(fmt.Sprintf("Not enough in the file to skip: ", err))
            return
        } else{
            llen -= int64(n)
            //fmt.Println("skipped ", n)
        }
    }
}

type sFChild interface {
    dump(indent string, in io.Reader);
}

var riffChildren map[[4]byte]sFChild = map[[4]byte]sFChild {
    [4]byte{'s','f','b','k'}:&root{},
};

type root struct{}
func (r *root) dump(indent string, in io.Reader) {
    fmt.Println(indent, "Got to riff root!")
    indent += "\t"
    for {
        ch, newRead := getChunkHeader(in)
        if ch == nil {
            break;
        }
        ch.dump(indent, newRead)
    }
}

var listChildren map[[4]byte]sFChild = map[[4]byte]sFChild {
    [4]byte{'I','N','F','O'}:info{},
    [4]byte{'s','d','t','a'}:sdta{},
    [4]byte{'p','d','t','a'}:pdta{},
};

func doLevel1Dump(desc string, indent string, in io.Reader) {
    fmt.Println(indent, "Got to ", desc, " section!")
    for {
        var sub [8]byte
        if n, err := io.ReadFull(in, sub[:]); n != 8 {
            fmt.Println(indent, "Got to end of ", desc, ". Error: ", err)
            return
        }
        llen := toint64(sub[4:8])
        ttype := [4]byte{sub[0],sub[1],sub[2],sub[3]}

        if ch, ok := subchunks[ttype]; ok {
            ch.dump(indent + "\t", io.LimitReader(in, llen))
        }else{
            fmt.Println(indent, "\tGot unknown subsection ", string(sub[0:4]), " of len ", llen, ". Skipping over...");
            skip(in, llen)
        }
    }

}

func readFullString(in io.Reader) string {
    var buf [1024]byte
    ret := ""
    for {
        n, err := in.Read(buf[:]);
        ret += string(buf[:n])
        if err != nil {
            return ret;
        }
    }
    panic("imposiburu!")
}

type info struct{}
func (i info) dump(indent string, in io.Reader) {
    doLevel1Dump("INFO", indent, in);
}

type sdta struct{}
func (s sdta) dump(indent string, in io.Reader) {
    doLevel1Dump("sdta", indent, in);
}

type pdta struct{}
func (p pdta) dump(indent string, in io.Reader) {
    doLevel1Dump("pdta", indent, in);
}

var subchunks map[[4]byte]sFChild = map[[4]byte]sFChild {
    [4]byte{'i','f','i','l'}:ifil{},
    [4]byte{'i','s','n','g'}:isng{},
    [4]byte{'I','N','A','M'}:inam{},
    [4]byte{'I','P','R','D'}:iprd{},
    [4]byte{'I','E','N','G'}:ieng{},
    [4]byte{'I','S','F','T'}:isft{},
    [4]byte{'I','C','R','D'}:icrd{},
    [4]byte{'I','C','M','T'}:icmt{},
    [4]byte{'I','C','O','P'}:icop{},

};

type ifil struct{}
func (i ifil) dump(indent string, in io.Reader) {
    var buf [4]byte
    if n, err := io.ReadFull(in, buf[:]); n != 4 {
        panic(fmt.Sprint("Couldn't read the ifil section: ", err))
    }
    major, minor := int(buf[0]) | (int(buf[1])<<8), int(buf[2]) | (int(buf[3])<<8)
    fmt.Println(indent, "Subchunk ifil: Version ", major, ".", minor)
}

type isng struct{}
func (i isng) dump(indent string, in io.Reader) {
    fmt.Println(indent, "Subchunk isng: Target Sound Engine: ", readFullString(in))
}

type inam struct{}
func (i inam) dump(indent string, in io.Reader) {
    fmt.Println(indent, "Subchunk INAM: Sound Font Bank Name: ", readFullString(in))
}

type iprd struct{}
func (i iprd) dump(indent string, in io.Reader) {
    fmt.Println(indent, "Subchunk IORD: Target product for sound bank: ", readFullString(in))
}

type ieng struct{}
func (i ieng) dump(indent string, in io.Reader) {
    fmt.Println(indent, "Subchunk IENG: Sound Designers and Engineers: ", readFullString(in))
}

type isft struct{}
func (i isft) dump(indent string, in io.Reader) {
    fmt.Println(indent, "Subchunk ISFT: Sound Font tools used: ", readFullString(in))
}

type icrd struct{}
func (i icrd) dump(indent string, in io.Reader) {
    fmt.Println(indent, "Subchunk ICRD: Creation date: ", readFullString(in))
}

type icmt struct{}
func (i icmt) dump(indent string, in io.Reader) {
    fmt.Println(indent, "Subchunk ICMT: Comments: ", readFullString(in))
}

type icop struct{}
func (i icop) dump(indent string, in io.Reader) {
    fmt.Println(indent, "Subchunk ICOP: Copyrights: ", readFullString(in))
}













