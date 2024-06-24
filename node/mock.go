package node

import "time"

var (
	rootNode = &Node{
		Id:           "/",
		Kind:         "FOLDER",
		Parents:      []string{},
		Status:       "AVAILABLE",
		CreatedBy:    "CloudDriveFiles",
		CreatedDate:  time.Now(),
		ModifiedDate: time.Now(),
		Version:      1,
		IsRoot:       true,
		Nodes: Nodes{
			"readme.md": &Node{
				Id:           "/README.md",
				Name:         "README.md",
				Kind:         "FILE",
				Parents:      []string{"/"},
				Status:       "AVAILABLE",
				CreatedBy:    "CloudDriveFiles",
				CreatedDate:  time.Now(),
				ModifiedDate: time.Now(),
				Version:      1,
				ContentProperties: ContentProperties{
					Version:     1,
					Extension:   "md",
					Size:        740,
					MD5:         "11c8fac0d43831697251fd0b869e77d7",
					ContentType: "text/plain",
					ContentDate: time.Now(),
				},
			},
			"pictures": &Node{
				Id:           "/pictures",
				Name:         "pictures",
				Kind:         "FOLDER",
				Parents:      []string{"/"},
				Status:       "AVAILABLE",
				CreatedBy:    "CloudDriveFiles",
				CreatedDate:  time.Now(),
				ModifiedDate: time.Now(),
				Version:      1,
				Nodes: Nodes{
					"logo.png": &Node{
						Id:           "/pictures/logo.png",
						Name:         "logo.png",
						Kind:         "FILE",
						Parents:      []string{"/pictures"},
						Status:       "AVAILABLE",
						CreatedBy:    "CloudDriveFiles",
						CreatedDate:  time.Now(),
						ModifiedDate: time.Now(),
						Version:      1,
						ContentProperties: ContentProperties{
							Version:     1,
							Extension:   "png",
							Size:        18750,
							MD5:         "c2c88b2bc3574122210c9f0cb45b0593",
							ContentType: "image/png",
							ContentDate: time.Now(),
						},
					},
				},
			},
		},
	}

	// Mocked is a valid tree (mock). The Ids are the fully-qualified path of
	// the file or folder to make testing easier.
	// /
	// |-- README.md
	// |-- pictures
	// |-- |
	//     | -- logo.png
	Mocked = &Tree{
		Node: rootNode,
		nodeIdMap: map[string]*Node{
			"/":                  rootNode,
			"/README.md":         rootNode.Nodes["readme.md"],
			"/pictures":          rootNode.Nodes["pictures"],
			"/pictures/logo.png": rootNode.Nodes["pictures"].Nodes["logo.png"],
		},
	}
)
