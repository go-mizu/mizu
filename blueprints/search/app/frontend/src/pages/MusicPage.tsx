import CategorySearchPage from './CategorySearchPage'
import { searchApi } from '../api/search'

export default function MusicPage() {
  return <CategorySearchPage category="music" tab="music" searchFn={searchApi.searchMusic} />
}
