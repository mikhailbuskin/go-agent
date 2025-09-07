import { useState } from 'react'

function App() {
  const [count, setCount] = useState(0)

  return (
    <>
      <button className='p-2 rounded bg-cyan-700 text-white' onClick={() => setCount((count) => count + 1)}>
        count is {count}
      </button>
    </>
  )
}

export default App
