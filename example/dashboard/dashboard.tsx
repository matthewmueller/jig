import React from "react"
import Layout from "../layout"

export type Props = {
  name: string
}

export default function Dashboard(props: Props) {
  const [count, setCount] = React.useState(0)
  return (
    <Layout title="Dashboard Page">
      <div>
        <h1>Welcome to the Dashboard, {props.name}.</h1>
        <p>Count: {count}</p>
        <button onClick={() => setCount(count + 1)}>Increment</button>
      </div>
    </Layout>
  )
}
