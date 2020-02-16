package main

type MangaRepository struct {
	mis []MangaInfo
	cis []ChapterInfo
	pis []PageInfo
}

func NewMangaRepository(mis []MangaInfo, cis []ChapterInfo, pis []PageInfo) *MangaRepository {
	return &MangaRepository{mis, cis, pis}
}

func (repo *MangaRepository) GetAllMangas() []MangaInfo {
	all := make([]MangaInfo, len(repo.mis))
	copy(all, repo.mis)
	return all
}

func (repo *MangaRepository) GetManga(id int) (MangaInfo, bool) {
	for _, mi := range repo.mis {
		if mi.ID == id {
			return mi, true
		}
	}

	return MangaInfo{}, false
}

func (repo *MangaRepository) GetChapter(chapterID int) (ChapterInfo, bool) {
	if chapterID >= 0 && chapterID < len(repo.cis) {
		return repo.cis[chapterID], true
	}

	return ChapterInfo{}, false
}

func (repo *MangaRepository) GetPageByID(id int) (PageInfo, bool) {
	if id >= 0 && id < len(repo.pis) {
		return repo.pis[id], true
	}

	return PageInfo{}, false
}
