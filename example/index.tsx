import React from "react"
import Layout from "./layout"

export type Props = {
  name: string
}

export default function Index(props: Props) {
  const [count, setCount] = React.useState(0)
  return (
    <div>
      <h1>Hello, {props.name}!</h1>
      <p>Count: {count}</p>
      <button onClick={() => setCount(count + 1)}>Increment</button>
    </div>
  )
}
