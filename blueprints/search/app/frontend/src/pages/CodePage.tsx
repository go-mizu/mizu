import CategorySearchPage from './CategorySearchPage'
import { searchApi } from '../api/search'

export default function CodePage() {
  return <CategorySearchPage category="code" tab="code" searchFn={searchApi.searchCode} />
}
