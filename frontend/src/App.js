import React, { Component } from 'react'
import './App.css'
import Chat from './Chat'

// import Map from './Map'
// import Tabs from './Tabs'
/*
 *
    return (
      <Tabs>
	<div label="Chat">
          <Chat />
	</div>
	<div label="Map">
	  <Map />
	</div>
	<div label="My IP">
	  My IP
	</div>
      </Tabs>
    )
    */

class App extends Component {
  render() {
    return (
          <Chat />
    )
  }
}

export default App

