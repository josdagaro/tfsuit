package rewrite

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strings"

    hcl "github.com/hashicorp/hcl/v2"
    "github.com/hashicorp/hcl/v2/hclsyntax"
    "github.com/sergi/go-diff/diffmatchpatch"

    "github.com/josdagaro/tfsuit/internal/config"
)

type Options struct{ Write, DryRun bool }

// ---------- helpers ----------
var nonAlnum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func toSnake(s string) string {
    s = strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(s), "_"), "_")
    return regexp.MustCompile(`_+`).ReplaceAllString(s, "_")
}

type rename struct{ Old, New string }

// ---------- main entry ----------
func Run(root string, cfg *config.Config, opt Options) error {
    files, err := collectTfFiles(root)
    if err != nil { return err }

    fileRen := map[string][]rename{}
    globalRen := map[string]string{}

    // pass 1 — build rename map
    for _, path := range files {
        src, _ := ioutil.ReadFile(path)
        file, diags := hclsyntax.ParseConfig(src, path, hcl.Pos{Line:1, Column:1})
        if diags.HasErrors() { continue }

        body := file.Body.(*hclsyntax.Body)
        for _, b := range body.Blocks {
            switch b.Type {
            case "variable", "output", "module":
                if len(b.Labels)==0 { continue }
                old := b.Labels[0]
                rule := map[string]*config.Rule{"variable":&cfg.Variables,"output":&cfg.Outputs,"module":&cfg.Modules}[b.Type]
                if rule.IsIgnored(old)||rule.Matches(old) { continue }
                newName := toSnake(old)
                fileRen[path]=append(fileRen[path],rename{old,newName}); globalRen[old]=newName
            case "resource":
                if len(b.Labels)<2 { continue }
                old:=b.Labels[1]
                if cfg.Resources.IsIgnored(old)||cfg.Resources.Matches(old){continue}
                newName:=toSnake(old)
                fileRen[path]=append(fileRen[path],rename{old,newName}); globalRen[old]=newName
            }
        }
    }
    if len(globalRen)==0 { fmt.Println("✅ No fixes needed"); return nil }

    // regex for cross‑references
    olds:=make([]string,0,len(globalRen));for o:=range globalRen{olds=append(olds,regexp.QuoteMeta(o))}
    sort.Slice(olds,func(i,j int)bool{return len(olds[i])>len(olds[j])})
    crossRe:=regexp.MustCompile(`([."\s])(`+strings.Join(olds,`|`)+`)(["\.\s])`)

    dmp:=diffmatchpatch.New()

    // pass 2 — rewrite files
    for _,path:=range files{
        orig,_:=ioutil.ReadFile(path)
        mod:=orig
        // local labels
        for _,rn:=range fileRen[path]{ mod=bytes.ReplaceAll(mod,[]byte(rn.Old),[]byte(rn.New)) }
        // cross refs
        mod=crossRe.ReplaceAllFunc(mod,func(b []byte)[]byte{
            m:=crossRe.FindSubmatch(b); if len(m)<4{return b}
            pre,name,suf:=m[1],string(m[2]),m[3]
            if nn,ok:=globalRen[name];ok{ return append(append(pre,[]byte(nn)...),suf...) }
            return b
        })
        if bytes.Equal(orig,mod){continue}

        if opt.DryRun {
            diffs:=dmp.DiffMain(string(orig),string(mod),false)
            fmt.Printf("\n--- %s\n%s",path,dmp.DiffPrettyText(diffs))
        } else if opt.Write {
            if err:=ioutil.WriteFile(path,mod,0o644);err!=nil{return err}
            fmt.Printf("fixed %s\n",path)
        }
    }
    return nil
}

// ---------- utils ----------
func collectTfFiles(root string)([]string,error){
    var out []string
    err:=filepath.Walk(root,func(p string,info os.FileInfo,err error)error{
        if err!=nil{return err}
        if info.IsDir(){ if info.Name()==".terraform"{return filepath.SkipDir}; return nil }
        if strings.HasSuffix(info.Name(),".tf"){ out=append(out,p) }
        return nil
    })
    return out,err
}
