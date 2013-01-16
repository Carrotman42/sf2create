
package sf2create

import (
    "fmt"
    "io"
)

// Reads a chunk header as a RIFF header plus four more bytes to get the type.
// It returns a new sFChild that represents the current section, a reader that
// is limited to the length of the section according to the header that was
// read, and the length of the section.
func getChunkHeader(in io.Reader) (sFChild, io.Reader, int64) {
    var header [12]byte;
    if n, _ := io.ReadFull(in, header[:]); n != 12 {
        return nil, nil, 0
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
        return ch, limited, llen
    }else if header[0] == 'L' && header[1] == 'I' && header[2] == 'S'&& header[3] == 'T'{
        ch,ok := listChildren[ttype]
        if !ok {
            panic (fmt.Sprint("Invalid LIST type: ", string(header[8:12])))
        }
        return ch, limited, llen
    }

    panic(fmt.Sprint("Invalid header type: '", string(header[:4]), "'"))
}

// Converts a slice of 4 bytes into an int64 using little-endian decoding
func toint64(dword []byte) int64 {
    return (int64(dword[3]) << 24) | (int64(dword[2]) << 16) | (int64(dword[1]) << 8) |int64(dword[0])
}

// Dumps the information of an SF2 compatible file from the provided reader into stdout
// Will panic if there was an error in the format of the file.
// Note: This reader should be buffered somehow, as many small reads will occur.
func Dump(in io.Reader) {
    child, newReader, llen := getChunkHeader(in);
    child.dump("", newReader, llen)
}

// Skips the given number of bytes on the reader, printing out an error message
// if there wasn't enough to read from the reader.
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

// Represents a node in the SF2 file. Right now it only requires support for
// dumping the file to stdout
type sFChild interface {
    // Dump any information retrievable for the section the sFChild represents
    // with the given indent
    dump(indent string, in io.Reader, llen int64);
}

// Represents all children that can be represented following a RIFF header
var riffChildren map[[4]byte]sFChild = map[[4]byte]sFChild {
    [4]byte{'s','f','b','k'}:root{},
};

type root struct{}
func (r root) dump(indent string, in io.Reader, llen int64) {
    fmt.Println(indent, "Got to riff root!")
    indent += "\t"
    for {
        ch, newRead, llen := getChunkHeader(in)
        if ch == nil {
            break;
        }
        ch.dump(indent, newRead, llen)
    }
}

// Represents all children that can be represented following a LIST header
// AKA "Level 1" nodes
var listChildren map[[4]byte]sFChild = map[[4]byte]sFChild {
    [4]byte{'I','N','F','O'}:info{},
    [4]byte{'s','d','t','a'}:sdta{},
    [4]byte{'p','d','t','a'}:pdta{},
};

// Provides common functionality for the Level 1 nodes
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
            ch.dump(indent + "\t", io.LimitReader(in, llen), llen)
        }else{
            fmt.Println(indent, "\tGot unknown subsection ", string(sub[0:4]), " of len ", llen, ". Skipping over...");
            skip(in, llen)
        }
    }

}

// Finds and returns a 0 terminated string in the provided bytes
func get0TermString(b []byte) string {
    var i int
    var v byte
    for i, v = range(b) {
        if v == 0 {
            break
        }
    }

    return string(b[:i])
}

// Reads as much as it can from a given reader, returning the data as a string
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
func (i info) dump(indent string, in io.Reader, llen int64) {
    doLevel1Dump("INFO", indent, in);
}

type sdta struct{}
func (s sdta) dump(indent string, in io.Reader, llen int64) {
    doLevel1Dump("sdta", indent, in);
}

type pdta struct{}
func (p pdta) dump(indent string, in io.Reader, llen int64) {
    doLevel1Dump("pdta", indent, in);
}

// Represents all Level 2 nodes. They are sorted here by the parent node
// to which they belong.
var subchunks map[[4]byte]sFChild = map[[4]byte]sFChild {
    // From the INFO chunk
    [4]byte{'i','f','i','l'}:ifil{},
    [4]byte{'i','s','n','g'}:isng{},
    [4]byte{'I','N','A','M'}:inam{},
    [4]byte{'I','P','R','D'}:iprd{},
    [4]byte{'I','E','N','G'}:ieng{},
    [4]byte{'I','S','F','T'}:isft{},
    [4]byte{'I','C','R','D'}:icrd{},
    [4]byte{'I','C','M','T'}:icmt{},
    [4]byte{'I','C','O','P'}:icop{},
    [4]byte{'i','r','o','m'}:irom{},
    [4]byte{'i','v','e','r'}:iver{},
    // From the pdta chunk
    [4]byte{'p','h','d','r'}:phdr{},
    [4]byte{'p','b','a','g'}:pbag{},
    [4]byte{'p','m','o','d'}:pmod{},
    [4]byte{'p','g','e','n'}:pgen{},
    [4]byte{'i','n','s','t'}:inst{},
    [4]byte{'i','b','a','g'}:ibag{},
    [4]byte{'i','m','o','d'}:imod{},
    [4]byte{'i','g','e','n'}:igen{},
    [4]byte{'s','h','d','r'}:shdr{},
};

// INFO chunk subchunks. Most of these deal with, as could be guessed,
// information about the SoundFont


func getVersion(in io.Reader) (major, minor uint16) {
    var buf [4]byte
    if n, err := io.ReadFull(in, buf[:]); n != 4 {
        panic(fmt.Sprint("Couldn't read the the version section correctly: ", err))
    }
    major, minor = uint16(buf[0]) | (uint16(buf[1])<<8), uint16(buf[2]) | (uint16(buf[3])<<8)
    return;
}

type ifil struct{}
func (i ifil) dump(indent string, in io.Reader, llen int64) {
    major, minor := getVersion(in);
    fmt.Println(indent, "Subchunk ifil: SoundFont Version ", major, ".", minor)
}

type iver struct{}
func (i iver) dump(indent string, in io.Reader, llen int64) {
    major, minor := getVersion(in);
    fmt.Println(indent, "Subchunk iver: Sound ROM Version ", major, ".", minor)
}

type isng struct{}
func (i isng) dump(indent string, in io.Reader, llen int64) {
    fmt.Println(indent, "Subchunk isng: Target Sound Engine: ", readFullString(in))
}

type inam struct{}
func (i inam) dump(indent string, in io.Reader, llen int64) {
    fmt.Println(indent, "Subchunk INAM: Sound Font Bank Name: ", readFullString(in))
}

type iprd struct{}
func (i iprd) dump(indent string, in io.Reader, llen int64) {
    fmt.Println(indent, "Subchunk IORD: Target product for sound bank: ", readFullString(in))
}

type ieng struct{}
func (i ieng) dump(indent string, in io.Reader, llen int64) {
    fmt.Println(indent, "Subchunk IENG: Sound Designers and Engineers: ", readFullString(in))
}

type isft struct{}
func (i isft) dump(indent string, in io.Reader, llen int64) {
    fmt.Println(indent, "Subchunk ISFT: Sound Font tools used: ", readFullString(in))
}

type icrd struct{}
func (i icrd) dump(indent string, in io.Reader, llen int64) {
    fmt.Println(indent, "Subchunk ICRD: Creation date: ", readFullString(in))
}

type icmt struct{}
func (i icmt) dump(indent string, in io.Reader, llen int64) {
    fmt.Println(indent, "Subchunk ICMT: Comments: ", readFullString(in))
}

type icop struct{}
func (i icop) dump(indent string, in io.Reader, llen int64) {
    fmt.Println(indent, "Subchunk ICOP: Copyrights: ", readFullString(in))
}

type irom struct{}
func (i irom) dump(indent string, in io.Reader, llen int64) {
    fmt.Println(indent, "Subchunk iver: Sound ROM Name: ", readFullString(in))
}


// From the pdta chunk. These deal with the contents of the SoundFont,
// including descriptions of instruments, generators, presets, and samples.

type phdr struct{}
func (i phdr) dump(indent string, in io.Reader, llen int64) {
    var buf [38]byte
    sl := buf[:]
    max := int(llen/38)
    fmt.Println(indent, "Section phdr (Preset Headers, count ", max, ")")
    indent += "\t"
    for i := 0; i < max; i++ {
        if n, err := io.ReadFull(in, sl); n != 38 {
            panic(fmt.Sprint("Couldn't read an entire header for phdr section: ", err))
        }
        presetName := string(get0TermString(buf[0:20]));
        preset := int(buf[20]) | (int(buf[21]) << 8)
        bank := int(buf[22]) | (int(buf[23]) << 8)
        presetBagIndex := int(buf[24]) | (int(buf[25]) << 8)
        library := toint64(buf[26:30])
        genre := toint64(buf[30:34])
        morphology := toint64(buf[34:38])
        fmt.Printf("%sName: %s; preset: %d; bank: %d; bag index: %d; library: %d, genre: %d, morphology: %d\n",
                    indent, presetName, preset, bank, presetBagIndex, library, genre, morphology)
    }
}

type pbag struct{}
func (i pbag) dump(indent string, in io.Reader, llen int64) {
    var buf [4]byte
    sl := buf[:]
    max := int(llen/4)
    fmt.Println(indent, "Section pbag (Preset Bag, count ", max, ")")
    indent += "\t"
    for i := 0; i < max; i++ {
        if n, err := io.ReadFull(in, sl); n != 4 {
            panic(fmt.Sprint("Couldn't read an entire header for pbag section: ", err))
        }
        genIndex, modIndex := int(buf[0]) | (int(buf[1]<<8)), int(buf[2]) | (int(buf[3])<<8)
        fmt.Printf("%sPreset bag index %d: genIndex: %d; modIndex: %d\n", indent, i, genIndex, modIndex)
    }
}

type pmod struct{}
func (i pmod) dump(indent string, in io.Reader, llen int64) {
    var buf [10]byte
    sl := buf[:]
    max := int(llen/10)
    fmt.Println(indent, "Section pmod (Preset Layer Modulators, count ", max, ")")
    indent += "\t"
    for i := 0; i < max; i++ {
        if n, err := io.ReadFull(in, sl); n != 10 {
            panic(fmt.Sprint("Couldn't read an entire header for pmod section: ", err))
        }
        modSrc := uint(buf[0]) | (uint(buf[1]<<8))
        modDest := uint(buf[2]) | (uint(buf[3]<<8))
        modAmt := uint16(int(buf[4]) | (int(buf[5]<<8)))
        modAmtSrcOper := int(buf[6]) | (int(buf[7]<<8))
        modTransOper := int(buf[8]) | (int(buf[9]<<8))
        fmt.Printf("%spmod index %d: Mod data source: %d; mod data dest: %d; mod amount: %d; modAmtSrcOper: %d; mod transform operation: %d\n",
                       indent, i, modSrc, modDest, modAmt, modAmtSrcOper, modTransOper)
    }
}

type pgen struct{}
func (i pgen) dump(indent string, in io.Reader, llen int64) {
    var buf [4]byte
    sl := buf[:]
    max := int(llen/4)
    fmt.Println(indent, "Section pgen (Preset Generators, count ", max, ")")
    indent += "\t"
    for i := 0; i < max; i++ {
        if n, err := io.ReadFull(in, sl); n != 4 {
            panic(fmt.Sprint("Couldn't read an entire header for pgen section: ", err))
        }
        genOper, amt := int(buf[0]) | (int(buf[1]<<8)), int(buf[2]) | (int(buf[3])<<8)
        fmt.Printf("%sPreset Generator index %d: genOper: %d; amt: %X\n", indent, i, genOper, amt)
    }
}

type inst struct{}
func (i inst) dump(indent string, in io.Reader, llen int64) {
    var buf [22]byte
    sl := buf[:]
    max := int(llen/22)
    fmt.Println(indent, "Section inst (Instruments, count ", max, ")")
    indent += "\t"
    for i := 0; i < max; i++ {
        if n, err := io.ReadFull(in, sl); n != 22 {
            panic(fmt.Sprint("Couldn't read an entire header for inst section: ", err))
        }
        instName := string(get0TermString(buf[0:20]));
        ibagIdx := uint16(int(buf[20]) | (int(buf[21])<<8))
        fmt.Printf("%sInstrument index %d: Name: %s; ibag index: %d\n", indent, i, instName, ibagIdx)
    }
}

type ibag struct{}
func (i ibag) dump(indent string, in io.Reader, llen int64) {
    var buf [4]byte
    sl := buf[:]
    max := int(llen/4)
    fmt.Println(indent, "Section ibag (Instrument Bag, count ", max, ")")
    indent += "\t"
    for i := 0; i < max; i++ {
        if n, err := io.ReadFull(in, sl); n != 4 {
            panic(fmt.Sprint("Couldn't read an entire header for ibag section: ", err))
        }
        igenIdx, imodIdx := int(buf[0]) | (int(buf[1]<<8)), int(buf[2]) | (int(buf[3])<<8)
        fmt.Printf("%sibag index %d: igen index: %d; imod index: %d\n", indent, i, igenIdx, imodIdx)
    }
}

type imod struct{}
func (i imod) dump(indent string, in io.Reader, llen int64) {
    var buf [10]byte
    sl := buf[:]
    max := int(llen/10)
    fmt.Println(indent, "Section imod (Preset Instrument Modulators, count ", max, ")")
    indent += "\t"
    for i := 0; i < max; i++ {
        if n, err := io.ReadFull(in, sl); n != 10 {
            panic(fmt.Sprint("Couldn't read an entire header for imod section: ", err))
        }
        modSrc := uint(buf[0]) | (uint(buf[1]<<8))
        modDest := uint(buf[2]) | (uint(buf[3]<<8))
        modAmt := uint16(int(buf[4]) | (int(buf[5]<<8)))
        modAmtSrcOper := int(buf[6]) | (int(buf[7]<<8))
        modTransOper := int(buf[8]) | (int(buf[9]<<8))
        fmt.Printf("%simod index %d: Mod data source: %d; mod data dest: %d; mod amount: %d; modAmtSrcOper: %d; mod transform operation: %d\n",
                       indent, i, modSrc, modDest, modAmt, modAmtSrcOper, modTransOper)
    }
}

type igen struct{}
func (i igen) dump(indent string, in io.Reader, llen int64) {
    var buf [4]byte
    sl := buf[:]
    max := int(llen/4)
    fmt.Println(indent, "Section igen (Instrument Generators, count ", max, ")")
    indent += "\t"
    for i := 0; i < max; i++ {
        if n, err := io.ReadFull(in, sl); n != 4 {
            panic(fmt.Sprint("Couldn't read an entire header for igen section: ", err))
        }
        genOper, amt := int(buf[0]) | (int(buf[1]<<8)), int(buf[2]) | (int(buf[3])<<8)
        fmt.Printf("%sInstrument Generator index %d: genOper: %d; amt: %X\n", indent, i, genOper, amt)
    }
}

type shdr struct{}
func (i shdr) dump(indent string, in io.Reader, llen int64) {
    var buf [46]byte
    sl := buf[:]
    max := int(llen/46)
    fmt.Println(indent, "Section shdr (Sample ids, count ", max, ")")
    indent += "\t"
    for i := 0; i < max; i++ {
        if n, err := io.ReadFull(in, sl); n != 46 {
            panic(fmt.Sprint("Couldn't read an entire header for shdr section: ", err))
        }
        sampleName := string(get0TermString(buf[0:20]));
        start, end := toint64(buf[20:24]), toint64(buf[24:28])
        lStart, lEnd := toint64(buf[28:32]), toint64(buf[32:36])
        sampleRate := toint64(buf[36:40])
        pitch, centsOff := buf[40], int8(buf[41])
        sampleLink := int(buf[42]) | (int(buf[43])<<8)
        sampleType := int(buf[44]) | (int(buf[45])<<8)
        fmt.Printf("%sSample index %d: Name: %s; start: %d; end: %d; loop start: %d; loop end: %d; sample rate: %d;\n" +
                    "%s\tpitch: %d; cents off: %d; sample link: %d; type: %d\n",
                    indent, i, sampleName, start, end, lStart, lEnd, sampleRate,
                    indent, pitch, centsOff, sampleLink, sampleType)
    }
}








