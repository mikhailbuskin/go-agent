import { useState } from 'react'

function Todo() {
  const [count, setCount] = useState(1);

  const message = {role:"assistant", content:"hi"};

  const [arr, setArr] = useState([message]);

  const [text, setText] = useState("");
  //arr.push(1);

  //const [arr, ] = [1,2,3]

  const handleDelete = (i) => {
    //debugger
    arr.splice(i,1);
    setArr([...arr]); // has an equals statement and checks if the arr == arr. This doesn't check what is inside the array but if it is the same array which it is so it doesn't rerender it. So you have to use ... and setArr(arr) doesn't work
  }

  const handleAdd = () => {
    const mes = message;
    mes.content = text;
    setArr([...arr, mes]);
    setText("");
  }

  //const setCount = () => (count) => count + 1;

  return (
    <>
      {/* <button className='p-2 rounded bg-cyan-700 text-white cursor-pointer' onClick={function() { setCount(count + 1) }}>
        count is {count}
      </button> */}
      {arr.map((ae,i) => {
        return (  // need to return because it is curly braces {}. With () it automatically returns
          <div>role: {ae.role}, content: {ae.content} 
            <button className='p-2 rounded bg-cyan-700 text-white cursor-pointer' onClick={function() {handleDelete(i)}}>
              delete
            </button> 
          </div>
        )
      })}

      <input id="id" type="text" className='border rounded' value={text} placeholder="Content" onChange={function(e) {setText(e.target.value)}}/>
      <button className='p-2 rounded bg-cyan-700 text-white cursor-pointer' onClick={function() { handleAdd() }}>
        add message 
      </button> 
    </>
  )
}

export default Todo
export function sum(a,b) {
  return a + b;
}