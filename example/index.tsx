import React from "react"

export type Props = {
  name: string
}

export default function Index(props: Props) {
  const [count, setCount] = React.useState(0)
  return (
    <main>
      <h1>Hello, {props.name}!</h1>
      <p>Count: {count}</p>
      <button onClick={() => setCount(count + 1)}>Increment</button>
    </main>
  )
}
