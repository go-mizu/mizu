import CategorySearchPage from './CategorySearchPage'
import { searchApi } from '../api/search'

export default function MapsPage() {
  return <CategorySearchPage category="maps" tab="maps" searchFn={searchApi.searchMaps} />
}
