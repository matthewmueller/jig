export type Props = {
  name: string;
};

export default function About(props: Props) {
  return (
    <main>
      <h1>About Page</h1>
      <p>This is the about page about {props.name}!!!!</p>
    </main>
  );
}
