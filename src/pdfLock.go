package main

import (
    "fmt"
    "os"
    "path/filepath"

    unicommon "github.com/unidoc/unidoc/common"
    pdfcore "github.com/unidoc/unidoc/pdf/core"
    "github.com/unidoc/unidoc/pdf/creator"
    pdf "github.com/unidoc/unidoc/pdf/model"
    "github.com/urfave/cli"
)

func init() {
    // Set debug-level logging via console.
    unicommon.SetLogger(unicommon.NewConsoleLogger(unicommon.LogLevelDebug))
}

//testEncrypt return if the pdf encrypted
//return true means the file is Encrypted,if errors happen the bool value is false
func testEncrypt(inputPath string) (bool, error) {
    f, err := os.Open(inputPath)
    if err != nil {
        return false, err
    }

    defer f.Close()

    pdfReader, err := pdf.NewPdfReader(f)
    if err != nil {
        return false, err
    }

    isEncrypted, err := pdfReader.IsEncrypted()
    if err != nil {
        return false, err
    }
    if isEncrypted {
        log.Infof("The PDF is already locked")
    }
    return true, err
}

// printAccessInfo
// inputPath the input file
// password the password specified
func printAccessInfo(inputPath string, password string) (error) {
    f, err := os.Open(inputPath)
    if err != nil {
        return err
    }

    defer f.Close()

    pdfReader, err := pdf.NewPdfReader(f)
    if err != nil {
        return err
    }

    canView, perms, err := pdfReader.CheckAccessRights([]byte(password))
    if err != nil {
        return err
    }

    if !canView {
        log.Infof("%s - Cannot view - No access with the specified password\n", inputPath)
        //return nil
    }

    log.Infof("Input file %s\n", inputPath)
    log.Infof("Access Permissions: %+v\n", perms)
    log.Infof("--------\n")

    // Print a text summary of the flags.
    booltext := map[bool]string{false: "No", true: "Yes"}
    log.Infof("Printing allowed? - %s\n", booltext[perms.Printing])
    if perms.Printing {
        log.Infof("Full print quality (otherwise print in low res)? - %s\n", booltext[perms.FullPrintQuality])
    }
    log.Infof("Modifications allowed? - %s\n", booltext[perms.Modify])
    log.Infof("Allow extracting graphics? %s\n", booltext[perms.ExtractGraphics])
    log.Infof("Can annotate? - %s\n", booltext[perms.Annotate])
    if perms.Annotate {
        log.Infof("Can fill forms? - Yes\n")
    } else {
        log.Infof("Can fill forms? - %s\n", booltext[perms.FillForms])
    }
    log.Infof("Extract text, graphics for users with disabilities? - %s\n", booltext[perms.DisabilityExtract])

    return nil
}

func addPassword(inputfilepath string, outputPath string, userPass string, ownerPass string) error {
    pdfWriter := pdf.NewPdfWriter()

    permissions := pdfcore.AccessPermissions{}
    // Allow printing with low quality
    permissions.Printing = false
    permissions.FullPrintQuality = false
    // Allow modifications.
    permissions.Modify = false
    // Allow annotations.
    permissions.Annotate = false
    permissions.FillForms = false
    // Allow modifying page order, rotating pages etc.
    permissions.RotateInsert = false
    // Allow extracting graphics.
    permissions.ExtractGraphics = false
    // Allow extracting graphics (accessibility)
    permissions.DisabilityExtract = false

    encryptOptions := &pdf.EncryptOptions{}
    encryptOptions.Permissions = permissions

    //err := pdfWriter.Encrypt([]byte(ownerPass+"A"), []byte(ownerPass+"B"), encryptOptions)
    err := pdfWriter.Encrypt([]byte(userPass), []byte(ownerPass), encryptOptions)
    if err != nil {
        return err
    }

    f, err := os.Open(inputfilepath)
    if err != nil {
        return err
    }

    defer f.Close()

    pdfReader, err := pdf.NewPdfReader(f)
    if err != nil {
        return err
    }

    isEncrypted, err := pdfReader.IsEncrypted()
    if err != nil {
        return err
    }
    if isEncrypted {
        return fmt.Errorf("The PDF is already locked (need to unlock first)")
    }

    numPages, err := pdfReader.GetNumPages()
    if err != nil {
        return err
    }

    for i := 0; i < numPages; i++ {
        pageNum := i + 1

        page, err := pdfReader.GetPage(pageNum)
        if err != nil {
            return err
        }

        err = pdfWriter.AddPage(page)
        if err != nil {
            return err
        }
    }

    fWrite, err := os.Create(outputPath)
    if err != nil {
        return err
    }

    defer fWrite.Close()

    err = pdfWriter.Write(fWrite)
    if err != nil {
        return err
    }

    return nil
}

// Watermark pdf file based on an image.
func addWatermarkImage(inputPath string, outputPath string, watermarkPath string) error {
    unicommon.Log.Debug("Input PDF: %v", inputPath)
    unicommon.Log.Debug("Watermark image: %s", watermarkPath)

    c := creator.New()

    watermarkImg, err := creator.NewImageFromFile(watermarkPath)
    if err != nil {
        return err
    }

    // Read the input pdf file.
    f, err := os.Open(inputPath)
    if err != nil {
        return err
    }
    defer f.Close()

    pdfReader, err := pdf.NewPdfReader(f)
    if err != nil {
        return err
    }

    numPages, err := pdfReader.GetNumPages()
    if err != nil {
        return err
    }

    for i := 0; i < numPages; i++ {
        pageNum := i + 1

        // Read the page.
        page, err := pdfReader.GetPage(pageNum)
        if err != nil {
            return err
        }

        // Add to creator.
        c.AddPage(page)

        watermarkImg.ScaleToWidth(c.Context().PageWidth)
        watermarkImg.SetPos(0, (c.Context().PageHeight-watermarkImg.Height())/2)
        watermarkImg.SetOpacity(0.2)

        _ = c.Draw(watermarkImg)
    }

    err = c.WriteToFile(outputPath)
    return err
}

func addWaterMarkAndEncryptedByConf(inputfile string) {
    outDir, outFilename := filepath.Split(inputfile)
    outputPath := filepath.Join(outDir, "Done_"+outFilename)
    watermarkFile := config.Watermark.Path
    addWatermarkImage(inputfile, outputPath, watermarkFile)
    userPass := config.Security.UserPass.Password2Add
    if config.Security.UserPass.Enable == false {
        userPass = ""
    }
    ownerPass := config.Security.OwnerPass.Password2Add
    if config.Security.OwnerPass.Enable == false {
        ownerPass = ""
    }
    //如果有一个需要加密则执行
    if config.Security.UserPass.Enable||config.Security.OwnerPass.Enable {
        addPassword(outputPath, outputPath, userPass, ownerPass)
    }

    err := printAccessInfo(inputfile, ownerPass)
    if err != nil {
        log.Errorf("Error: %v\n", err)
    }

}


//depreciated
func dopdf2(inputfile string, pass string) {

    err := printAccessInfo(inputfile, pass)
    outdir, outfilename := filepath.Split(inputfile)
    outputPath := filepath.Join(outdir, "locked_"+outfilename)
    outputPath2 := filepath.Join(outdir, "watermarked_"+outfilename)
    log.Info("outName:", outputPath)
    addWatermarkImage(inputfile, outputPath2, "./aaa.png")
    addPassword(outputPath2, outputPath, pass, pass)
    if err != nil {
        log.Errorf("Error: %v\n", err)
    }
}

func cliWaterMarkAndEncrypt() cli.Command {
    command := cli.Command{
        Name:        "unidoc",
        Aliases:     []string{"p"},
        Category:    "Common tools",
        Usage:       "Check a exe is 32bit or 64 bit",
        UsageText:   "Example: mo exe64 c:\a.exe ",
        Description: "Check a exe is 32bit or 64 bit.",
        ArgsUsage:   "<filename>",
        //Flags: []cli.Flag{
        //    cli.BoolFlag{
        //        Name:   "show,s",
        //        Usage:  "show current password",
        //        Hidden: true,
        //    },
        //},
        Action: func(c *cli.Context) error {
            if c.NArg() < 2 {
                log.Fatal("Args' number less than 2")
            }
            dopdf2(c.Args().First(), c.Args().Get(1))
            //calcBase(c.Args().First(), c.Bool("debase"))
            return nil
        },
    }
    return command
}
