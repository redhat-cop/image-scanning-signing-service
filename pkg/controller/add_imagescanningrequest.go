package controller

import (
	"github.com/redhat-cop/image-security/pkg/controller/imagescanningrequest"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, imagescanningrequest.Add)
}
