// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ucloud"
)

func init() {
	type ImageListOptions struct {
		Owner string `help:"Owner type" choices:"Base|Business|Custom"`
		Id    string `help:"Image ID"`
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *ucloud.SRegion, args *ImageListOptions) error {
		images, e := cli.GetImages(args.Owner, args.Id)
		if e != nil {
			return e
		}
		printList(images, 0, 0, 0, []string{})
		return nil
	})

	type ImageDeleteOptions struct {
		ID string `help:"ID or Name to delete"`
	}
	shellutils.R(&ImageDeleteOptions{}, "image-delete", "Delete image", func(cli *ucloud.SRegion, args *ImageDeleteOptions) error {
		return cli.DeleteImage(args.ID)
	})

	shellutils.R(&ImageDeleteOptions{}, "image-show", "Show image", func(cli *ucloud.SRegion, args *ImageDeleteOptions) error {
		img, err := cli.GetImage(args.ID)
		if err != nil {
			return err
		}
		printObject(img)
		return nil
	})
}
