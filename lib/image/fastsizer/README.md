# fastimage

Detect the size and type of an image, quickly.

Finds the type and/or size of an image given its uri by fetching as little as needed.

fastimage only reads the minimal amount needed to return the dimensions.
It does not interpret any more of the image than necessary, and only reads the
minimal amount.

Unlike the original libraries, it does not do any fetching of the resource. It
wants to be given an io.Reader instead, to allow generic detection from any source.

## Usage

For instance, this is a big 10MB JPEG image on wikipedia:

**Method1**

```go
fs := fastsizer.NewFastSizer()
typ, sz, err := fs.Detect(myReader)
```

## Supported file types

| File type | Can detect type? | Can detect size? |
|-----------|:----------------:|:----------------:|
| PNG       | Yes              | Yes              |
| JPEG      | Yes              | Yes              |
| GIF       | Yes              | Yes              |
| BMP       | Yes              | Yes              |
| TIFF      | Yes              | Yes              |
| WEBP      | Yes              | Yes              |

### License

fastimage is under MIT license. See the LICENSE file for details.

Based on code from [@sillydong](https://github.com/sillydong/fastimage), which
in turn was originally based on code from [Ruben Fonseca](https://github.com/rubenfonseca/fastimage).
