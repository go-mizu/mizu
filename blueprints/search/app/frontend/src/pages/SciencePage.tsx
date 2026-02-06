import CategorySearchPage from './CategorySearchPage'
import { searchApi } from '../api/search'

export default function SciencePage() {
  return <CategorySearchPage category="science" tab="science" searchFn={searchApi.searchScience} />
}
