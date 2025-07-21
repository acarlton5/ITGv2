<p align="center">
<a  href="https://github.com/GRVYDEV/ITG-react">
    <img src="images/itglogo.svg" alt="Logo" width="150" height="150">
</a>
</p>
  <h1 align="center">Project ITG React</h1>
<div align="center">
  <a href="https://github.com/GRVYDEV/ITG-react/stargazers"><img src="https://img.shields.io/github/stars/GRVYDEV/ITG-react" alt="Stars Badge"/></a>
<a href="https://github.com/GRVYDEV/ITG-react/network/members"><img src="https://img.shields.io/github/forks/GRVYDEV/ITG-react" alt="Forks Badge"/></a>
<a href="https://github.com/GRVYDEV/ITG-react/pulls"><img src="https://img.shields.io/github/issues-pr/GRVYDEV/ITG-react" alt="Pull Requests Badge"/></a>
<a href="https://github.com/GRVYDEV/ITG-react/issues"><img src="https://img.shields.io/github/issues/GRVYDEV/ITG-react" alt="Issues Badge"/></a>
<a href="https://github.com/GRVYDEV/ITG-react/graphs/contributors"><img alt="GitHub contributors" src="https://img.shields.io/github/contributors/GRVYDEV/ITG-react?color=2b9348"></a>
<a href="https://github.com/GRVYDEV/ITG-react/blob/master/LICENSE"><img src="https://img.shields.io/github/license/GRVYDEV/ITG-react?color=2b9348" alt="License Badge"/></a>
</div>
<br />
<p align="center">
  <p align="center">
    A React website that connects to ITG WebRTC via a websocket to negotiate SDPs and display a WebRTC stream.
    <!-- <br /> -->
    <!-- <a href="https://github.com/GRVYDEV/ITG-react"><strong>Explore the docs »</strong></a> -->
    <br />
    <br />
    <a href="https://youtu.be/Dzin4_A8RDs">View Demo</a>
    ·
    <a href="https://github.com/GRVYDEV/ITG-react/issues">Report Bug</a>
    ·
    <a href="https://github.com/GRVYDEV/ITG-react/issues">Request Feature</a>
  </p>
</p>

<!-- TABLE OF CONTENTS -->
<details open="open">
  <summary><h2 style="display: inline-block">Table of Contents</h2></summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
      <ul>
        <li><a href="#built-with">Built With</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
    <li><a href="#contact">Contact</a></li>
    <li><a href="#acknowledgements">Acknowledgements</a></li>
  </ol>
</details>

<!-- ABOUT THE PROJECT -->

## About The Project

<!-- [![Product Name Screen Shot][product-screenshot]](https://example.com) -->

This is one of three components required for Project ITG. Project ITG is a fully self contained live streaming server. With this you will be able to deploy your own sub-second latency live streaming platform. This particular repository connects via websocket to ITG WebRTC and displays a WebRTC stream. In order for this to work the Project ITG WebRTC and Project ITG Ingest are required. 

### Built With

- React

### Dependencies

- [ITG WebRTC](https://github.com/GRVYDEV/ITG-webrtc)
- [ITG Ingest](https://github.com/GRVYDEV/ITG-ingest)

<!-- GETTING STARTED -->

## Getting Started

## Setup

### Docker

1. Install [git](https://git-scm.com/downloads)
1. Build the image from the master branch with:

    ```sh
    docker build -t grvydev/itg-react https://github.com/GRVYDEV/ITG-react.git
    ```

1. Run it with

    ```sh
    docker run -it --rm \
      -p 8000:80/tcp \
      -e WEBSOCKET_HOST=localhost \
      -e WEBSOCKET_PORT=8080 \
      grvydev/itg-react
    ```

    Where your websocket host from the browser/client perspective is accessible on `localhost:8080`.

1. You can now access it at [localhost:8000](http://localhost:8000).

### Locally

To get a local copy up and running follow these simple steps.

#### Prerequisites

In order to run this npm is required. Installation instructions can be found <a href="https://www.rust-lang.org/tools/https://www.npmjs.com/get-npm">here</a>. Npm Serve is required as well if you want to host this on your machine. That can be found <a href="https://www.npmjs.com/package/serve">here</a>

#### Installation

```sh
git clone https://github.com/GRVYDEV/ITG-react.git
cd ITG-react
npm install
```

<!-- USAGE EXAMPLES -->

#### Usage

First build the frontend

```sh
cd ITG-react
# If the build fails with an OpenSSL error (common on Node 17+)
export NODE_OPTIONS=--openssl-legacy-provider
npm run build
```

You should then configure the websocket URL in `config.json` in the `build` directory.

Now you can host the static site locally, by using `serve` for example

```sh
serve -s build -l 80
```

This will serve the build folder on port 80 of your machine meaning it can be retrieved via a browser by either going to your machines public IP or hostname

<!-- _For more examples, please refer to the [Documentation](https://example.com)_ -->

<!-- ROADMAP -->

## Roadmap

See the [open issues](https://github.com/GRVYDEV/ITG-react/issues) for a list of proposed features (and known issues).

<!-- CONTRIBUTING -->

## Contributing

Contributions are what make the open source community such an amazing place to be learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

<!-- LICENSE -->

## License

Distributed under the MIT License. See `LICENSE` for more information.

<!-- CONTACT -->

## Contact

Garrett Graves - [@grvydev](https://twitter.com/grvydev)

Project Link: [https://github.com/GRVYDEV/ITG-react](https://github.com/GRVYDEV/ITG-react)

<!-- ACKNOWLEDGEMENTS -->

## Acknowledgements

- [Sean Dubois](https://github.com/Sean-Der)


<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->


