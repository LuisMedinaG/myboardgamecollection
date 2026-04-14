import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import CollectionPage from './pages/CollectionPage'
import GameDetailPage from './pages/GameDetailPage'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<CollectionPage />} />
        <Route path="games/:id" element={<GameDetailPage />} />
      </Route>
    </Routes>
  )
}
