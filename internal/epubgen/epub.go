package epubgen

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bmaupin/go-epub"
	"github.com/go-shiori/go-readability"
	"github.com/gosimple/slug"
	"github.com/ryan-gang/kindle-send-daemon/internal/config"
	"github.com/ryan-gang/kindle-send-daemon/internal/util"
)

type epubmaker struct {
	Epub      *epub.Epub
	downloads map[string]string
}

func NewEpubmaker(title string) *epubmaker {
	downloadMap := make(map[string]string)
	return &epubmaker{
		Epub:      epub.NewEpub(title),
		downloads: downloadMap,
	}
}

func fetchReadable(url string) (readability.Article, error) {
	return readability.FromURL(url, 30*time.Second)
}

// Point remote image link to downloaded image
func (e *epubmaker) changeRefs(i int, img *goquery.Selection) {
	img.RemoveAttr("loading")
	img.RemoveAttr("srcset")
	imgSrc, exists := img.Attr("src")
	if exists {
		if _, ok := e.downloads[imgSrc]; ok {
			util.Green.Printf("Setting img src from %s to %s \n", imgSrc, e.downloads[imgSrc])
			img.SetAttr("src", e.downloads[imgSrc])
		}
	}
}

// compressImage compresses the image data based on its type
func compressImage(imgData []byte, imgURL string) ([]byte, error) {
	// Determine image type from URL or content
	imgType := strings.ToLower(filepath.Ext(imgURL))

	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	// Compress based on format
	if format == "jpeg" || imgType == ".jpg" || imgType == ".jpeg" {
		// JPEG quality settings (0-100), 85 is a good balance
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	} else if format == "png" || imgType == ".png" {
		// For PNG, using default compression
		encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
		err = encoder.Encode(&buf, img)
	} else {
		// For other formats, just return the original data
		return imgData, nil
	}

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Download images and add to epub zip
func (e *epubmaker) downloadImages(i int, img *goquery.Selection) {
	util.CyanBold.Println("Downloading Images")
	imgSrc, exists := img.Attr("src")

	if exists {
		// don't download same thing twice
		if _, ok := e.downloads[imgSrc]; ok {
			return
		}

		// pass unique and safe image names here, then it will not crash on windows
		// use murmur hash to generate file name
		imageFileName := util.GetHash(imgSrc)

		// Download the image data first
		resp, err := http.Get(imgSrc)
		if err != nil {
			util.Red.Printf("Couldn't download image %s: %s\n", imgSrc, err)
			return
		}
		defer resp.Body.Close()

		// Read the image data
		imgData, err := io.ReadAll(resp.Body)
		if err != nil {
			util.Red.Printf("Error reading image data from %s: %s\n", imgSrc, err)
			return
		}

		// Compress the image
		compressedImgData, err := compressImage(imgData, imgSrc)
		if err != nil {
			util.Red.Printf("Error compressing image %s: %s\n", imgSrc, err)
			// Fallback to original image if compression fails
			compressedImgData = imgData
		} else {
			originalSize := len(imgData)
			compressedSize := len(compressedImgData)
			reduction := float64(originalSize-compressedSize) / float64(originalSize) * 100
			util.Green.Printf("Image %s compressed: %d KB → %d KB (%.1f%% reduction)\n",
				filepath.Base(imgSrc), originalSize/1024, compressedSize/1024, reduction)
		}

		// Create a temp file for the compressed image
		tempFile, err := os.CreateTemp("", "kindle-send-img-*"+filepath.Ext(imageFileName))
		if err != nil {
			util.Red.Printf("Couldn't create temp file for image %s: %s\n", imgSrc, err)
			return
		}
		defer tempFile.Close()

		// Write compressed data to temp file
		if _, err := tempFile.Write(compressedImgData); err != nil {
			util.Red.Printf("Error writing to temp file for image %s: %s\n", imgSrc, err)
			os.Remove(tempFile.Name())
			return
		}

		// Create a custom URL for the temp file to use with AddImage
		tempFileURL := "file://" + tempFile.Name()

		// Add the compressed image to epub using the URL approach
		imgRef, err := e.Epub.AddImage(tempFileURL, imageFileName)
		if err != nil {
			util.Red.Printf("Couldn't add image %s : %s\n", imgSrc, err)
			os.Remove(tempFile.Name())
			return
		}

		// Clean up temp file after it's added to the epub
		defer os.Remove(tempFile.Name())

		util.Green.Printf("Downloaded and compressed image %s\n", imgSrc)
		e.downloads[imgSrc] = imgRef
	}
}

// Fetches images in article and then embeds them into epub
func (e *epubmaker) embedImages(wg *sync.WaitGroup, article *readability.Article) {
	util.Cyan.Println("Embedding images in ", article.Title)
	defer wg.Done()
	// Compression is now handled in downloadImages function
	doc := goquery.NewDocumentFromNode(article.Node)

	//download all images
	doc.Find("img").Each(e.downloadImages)

	//Change all refs, doing it in two phases to download repeated images only once
	doc.Find("img").Each(e.changeRefs)

	content, err := doc.Html()

	if err != nil {
		util.Red.Printf("Error converting modified %s to HTML, it will be transferred without images : %s \n", article.Title, err)
	} else {
		article.Content = content
	}
}

// TODO: Look for better formatting, this is bare bones
func prepare(article *readability.Article) string {
	return "<h1>" + article.Title + "</h1>" + article.Content
}

// Add articles to epub
func (e *epubmaker) addContent(articles *[]readability.Article) error {
	added := 0
	for _, article := range *articles {
		_, err := e.Epub.AddSection(prepare(&article), article.Title, "", "")
		if err != nil {
			util.Red.Printf("Couldn't add %s to epub : %s", article.Title, err)
		} else {
			added++
		}
	}
	util.Green.Printf("Added %d articles\n", added)
	if added == 0 {
		return errors.New("no article was added, epub creation failed")
	}
	return nil
}

// Make : Generates a single epub from a slice of urls, returns file path
func Make(pageUrls []string, title string) (string, error) {
	//TODO: Parallelize fetching pages

	//Get readable article from urls
	readableArticles := make([]readability.Article, 0)
	for _, pageUrl := range pageUrls {
		article, err := fetchReadable(pageUrl)
		if err != nil {
			util.Red.Printf("Couldn't convert %s because %s", pageUrl, err)
			util.Magenta.Println("SKIPPING ", pageUrl)
			continue
		}
		util.Green.Printf("Fetched %s --> %s\n", pageUrl, article.Title)
		readableArticles = append(readableArticles, article)
	}

	if len(readableArticles) == 0 {
		return "", errors.New("no readable url given, exiting without creating epub")
	}

	if len(title) == 0 {
		title = readableArticles[0].Title
		util.Magenta.Printf("No title supplied, inheriting title of first readable article : %s \n", title)
	}

	book := NewEpubmaker(title)

	//get images and embed them
	var wg sync.WaitGroup

	for i := 0; i < len(readableArticles); i++ {
		wg.Add(1)
		go book.embedImages(&wg, &readableArticles[i])
	}

	wg.Wait()

	err := book.addContent(&readableArticles)
	if err != nil {
		return "", err
	}
	var storeDir string
	if len(config.GetInstance().StorePath) == 0 {
		storeDir, err = os.Getwd()
		if err != nil {
			util.Red.Println("Error getting current directory, trying fallback")
			storeDir = "./"
		}
	} else {
		storeDir = config.GetInstance().StorePath
	}

	titleSlug := slug.Make(title)
	var filename string
	if len(titleSlug) == 0 {
		filename = "kindle-send-doc-" + util.GetHash(readableArticles[0].Content) + ".epub"
	} else {
		filename = titleSlug + ".epub"
	}
	filepath := path.Join(storeDir, filename)
	err = book.Epub.Write(filepath)
	if err != nil {
		return "", err
	}
	return filepath, nil
}
