package libsquash

// SquashOptions are the additional options that may be used to modify the
// squash process
type SquashOptions struct {
	// From is the "start" layer for the squash process. The squash layer will
	// contain the filesystem of the From layer and all of its children but
	// none of its parents.  If not provided, From is defaulted to the image root.
	From string

	// Tags is a list of tag/repo mappings that will be applied to the final
	// ImageID.  This is done by writing the "repositories" file into the final
	// image tarball.  If this option is not provided, the image can still be
	// tagged by using the docker tag api and the final image ID which is
	// written to the imageIDOut io.Writer that gets passed into Squas
	Tags TagList
}

// Repository is a wrapper type that exists to help clarify the structure of
// the Repositories type.  It represents the repository part of an image name
// (e.g. for busybox:latest, it is the "busybox" part)
type Repository string

// Tag is a wrapper type that exists to help clarify the structure of
// the Repositories type.  It represents the tag part of an image name
// (e.g. for busybox:latest, it is the "latest" part)
type Tag string

// ImageID is a wrapper type that exists to help clarify the structure of the
// Repositories type.  It represents the id of the image that is being tagged
// with a given repo and tag
type ImageID string

// Repositories is a type for representing the structure of the "repositories"
// file that can be written into the image tarball to produce tags for the
// image when the tarball is loaded
type Repositories map[Repository]map[Tag]ImageID

// TagList is a type for representing the repo-tag mapping that will be
// converted into a Repositories instance, given an image ID
type TagList map[Repository][]Tag

// ProduceRepositories produces a Repositories instance that will get written
// into the "repositories" file in the image tarball.  All of the tags in tl
// will get applied to the same imageID
func (tl TagList) ProduceRepositories(imageID string) Repositories {
	ret := map[Repository]map[Tag]ImageID{}
	for repo, tagArr := range tl {
		if ret[repo] == nil {
			ret[repo] = map[Tag]ImageID{}
		}
		if len(tagArr) == 0 {
			ret[repo][Tag("latest")] = ImageID(imageID)
		} else {
			for _, tag := range tagArr {
				ret[repo][Tag(tag)] = ImageID(imageID)
			}
		}
	}
	return ret
}
