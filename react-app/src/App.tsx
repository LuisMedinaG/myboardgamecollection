import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import CollectionPage from './pages/CollectionPage'
import GameDetailPage from './pages/GameDetailPage'
import VibesPage from './pages/VibesPage'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<CollectionPage />} />
        <Route path="games/:id" element={<GameDetailPage />} />
        <Route path="vibes" element={<VibesPage />} />
      </Route>
    </Routes>
  )
}
