import Layout from "./layout"

export type Props = {
  name: string
}

export default function About(props: Props) {
  return (
    <Layout title="About Page">
      <div>
        <h1>About Page</h1>
        <p>This is the about page about {props.name}!</p>
      </div>
    </Layout>
  )
}
