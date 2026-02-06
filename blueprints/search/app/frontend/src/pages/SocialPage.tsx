import CategorySearchPage from './CategorySearchPage'
import { searchApi } from '../api/search'

export default function SocialPage() {
  return <CategorySearchPage category="social" tab="social" searchFn={searchApi.searchSocial} />
}
