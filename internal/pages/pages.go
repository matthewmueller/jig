package pages

type Page struct {
	EntryPath  string
	LayoutPath *string
	ErrorPath  *string
}

func Find(dir string, entries ...string) ([]*Page, error) {
	layout := "layout.tsx"
	errorPath := "error.tsx"
	return []*Page{
		{
			EntryPath:  "index.tsx",
			LayoutPath: &layout,
			ErrorPath:  &errorPath,
		},
		{
			EntryPath:  "about.tsx",
			LayoutPath: &layout,
			ErrorPath:  &errorPath,
		},
		{
			EntryPath:  "dashboard/dashboard.tsx",
			LayoutPath: &layout,
			ErrorPath:  &errorPath,
		},
	}, nil
	// pages := make([]*Page, 0, len(entries))
	// eg := new(errgroup.Group)
	// for _, entry := range entries {
	// 	eg.Go(func() error {
	// 		page, err := findPage(dir, dir, entry)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		pages = append(pages, page)
	// 		return nil
	// 	})
	// }
	// return nil, eg.Wait()
}

// func findPage(root, dir, entry string) (*Page, error) {
// 	fmt.Println("Finding page in root:", root, "with entry:", entry)
// 	return nil, nil
// }
