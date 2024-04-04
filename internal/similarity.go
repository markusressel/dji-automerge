package internal

import "github.com/vitali-fedulov/images4"

type Similarity struct {
	PropMetric            float64
	ProportionsPercentage float64

	Y         float64
	Ypercent  float64
	Cb        float64
	CbPercent float64
	Cr        float64
	CrPercent float64
}

func (s Similarity) Similar() bool {
	propSimilar := s.PropMetric <= thresholdProp
	if !propSimilar {
		return false
	}
	eucSimilar := s.Y < thresholdY && // Luma as most sensitive.
		s.Cb < thresholdCbCr &&
		s.Cr < thresholdCbCr
	return eucSimilar
}

func compareImages(imagePath1, imagePath2 string) (Similarity, error) {
	// Opening and decoding images. Silently discarding errors.
	img1, err := images4.Open(imagePath1)
	if err != nil {
		return Similarity{}, err
	}
	img2, err := images4.Open(imagePath2)
	if err != nil {
		return Similarity{}, err
	}

	// Icons are compact hash-like image representations.
	iconA := images4.Icon(img1)
	iconB := images4.Icon(img2)

	// Comparison. Images are not used directly.
	// Use func CustomSimilar for different precision.

	propMetric := images4.PropMetric(iconA, iconB)
	proportionsPercentage := propMetric / thresholdProp

	m1, m2, m3 := images4.EucMetric(iconA, iconB)

	mp1 := m1 / thresholdY
	mp2 := m2 / thresholdCbCr
	mp3 := m3 / thresholdCbCr

	return Similarity{
		PropMetric:            propMetric,
		ProportionsPercentage: proportionsPercentage,

		Y:         m1,
		Ypercent:  mp1,
		Cb:        m2,
		CbPercent: mp2,
		Cr:        m3,
		CrPercent: mp3,
	}, nil
}
