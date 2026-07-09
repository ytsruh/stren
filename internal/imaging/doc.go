// Package imaging decodes, resizes, and re-encodes images in a uniform way.
//
// The package wraps the Go standard library's image decoders (JPEG, PNG)
// with the high-quality draw.CatmullRom scaler from golang.org/x/image.
// It is intentionally small: one Processor interface, one implementation,
// a handful of sentinel errors. The intent is to be the single place
// every image uploaded to the app flows through, so output dimensions,
// aspect ratio handling, and output format stay consistent.
//
// Typical usage:
//
//	proc := imaging.NewStdProcessor()
//	display, err := proc.Process(ctx, src, imaging.ProcessOptions{
//	    TargetWidth:  800,
//	    TargetHeight: 600,
//	    Quality:      85,
//	    Format:       imaging.FormatJPEG,
//	})
//	if err != nil {
//	    return err
//	}
//	// upload display.Data to R2
package imaging
