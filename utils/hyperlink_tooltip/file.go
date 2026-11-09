package hyperlink_tooltip

import "path"

func file(link *linkT) {
	link.Append("Path", "f07b", path.Dir(link.Target))
	link.Append("File", "f15b", path.Base(link.Target))
}
