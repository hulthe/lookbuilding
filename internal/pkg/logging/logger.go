package logging

import "github.com/sirupsen/logrus"

var (
	Logger logrus.Logger = *logrus.New()
)