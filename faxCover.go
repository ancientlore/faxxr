package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/myesui/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

type faxCoverDetails struct {
	FromPhone string
	FromName  string
	FromAddr1 string
	FromAddr2 string
	ToPhone   string
	ToName    string
	Subject   string
	Text      string
	Quality   string
	ImageFile string // if provided, an extra page with the image is written
	pdfFile   string
	faxSID    string
	created   time.Time
}

func faxText(pdf *gofpdf.Fpdf, text string, bold bool, size float64) {
	b := ""
	if bold {
		b = "B"
	}
	pdf.SetFont("Arial", b, size)
	pdf.SetX(72)
	pdf.Write(size+2, text)
	pdf.Ln(size + 8)
}

func faxImagePage(pdf *gofpdf.Fpdf, fileName string, x, y, w, h float64) {
	pdf.AddPage()
	pdf.ImageOptions(
		fileName,
		x,
		y,
		w,
		h,
		true,
		gofpdf.ImageOptions{
			ImageType: "",
			ReadDpi:   true,
		},
		0,
		"",
	)
}

func faxCover(tmpDir string, details *faxCoverDetails) (string, error) {
	pdf := gofpdf.New("Portrait", "pt", "Letter", "")
	pdf.SetTitle("Fax", true)

	pdf.AddPage()
	pdf.SetMargins(72, 72, 72)
	pdf.SetY(76)

	tzEST, err := time.LoadLocation("America/New_York")
	if err != nil {
		tzEST = time.Local
	}

	faxText(pdf, " FAX", true, 32)
	faxText(pdf, time.Now().In(tzEST).Format(time.RFC850), false, 10)
	faxText(pdf, "\nFROM:", false, 12)
	faxText(pdf, details.FromName, true, 16)
	if details.FromAddr1 != "" {
		faxText(pdf, details.FromAddr1, false, 14)
	}
	if details.FromAddr2 != "" {
		faxText(pdf, details.FromAddr2, false, 14)
	}
	faxText(pdf, details.FromPhone, false, 14)
	faxText(pdf, "\nTO:", false, 12)
	faxText(pdf, details.ToName, true, 16)
	faxText(pdf, details.ToPhone, false, 14)
	faxText(pdf, "\nREGARDING:", false, 12)
	faxText(pdf, details.Subject, true, 16)
	if details.Text != "" {
		faxText(pdf, details.Text, false, 14)
	}
	faxText(pdf, "\n~ Sent by github.com/ancientlore/faxxr ~", false, 8)
	pdf.Image("media/m.png", 7*72, 76, 32, 32, false, "", 0, "")
	pdf.Rect(72, 72, 6.5*72, 36, "D")

	// You can attach an image directly while generating the cover
	if details.ImageFile != "" {
		faxImagePage(pdf, details.ImageFile, 72, 72, 6.5*72, 0)
	}

	fileStr := filepath.Join(tmpDir, uuid.NewV4().String()+".pdf")
	err = pdf.OutputFileAndClose(fileStr)
	return fileStr, err
}

func mergePdfs(tmpDir string, files []string) (string, error) {
	outfile := filepath.Join(tmpDir, uuid.NewV4().String()+".pdf")
	config := pdfcpu.NewDefaultConfiguration()
	config.ValidationMode = pdfcpu.ValidationRelaxed
	config.Reader15 = true
	config.WriteXRefStream = true
	config.WriteObjectStream = true
	err := api.MergeCreateFile(files, outfile, config)
	// delete old files
	for _, f := range files {
		err2 := os.Remove(f)
		if err2 != nil {
			log.Print(err2)
		}
	}
	return outfile, err
}
