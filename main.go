package main

import (
    "./sf2create"
   // "fmt"
    "os"
    "bufio"
)

func main() {
    f, err := os.Open("sf.sf2")
    if err != nil {
        panic(err.Error());
    }
    defer f.Close()
    buf := bufio.NewReader(f)
    sf2create.Dump(buf);
}
