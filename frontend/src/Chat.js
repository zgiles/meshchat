import React, { Component } from 'react'
import ChatInput from './ChatInput'
import ChatMessage from './ChatMessage'

// const URL = 'ws://localhost:8080/ws'
const URL = 'ws://' + window.location.hostname + (window.location.port ? ':'+window.location.port : '' ) + '/ws';
const i = 1000

var green = 3000;
var yellow = 4000;
var red = 5000;

class Chat extends Component {
  state = {
    name: 'User',
    id: '',
    messages: [],
    users: {},
    connected: false,
  }

  // ws = new WebSocket(URL)
  ws = null
  shouldreconnect = false
  userinterval = null

  websocketsetup(s) {
    s.onopen = () => {
      console.log('connected')
      this.setState({ connected: true })
      this.sendPresence()
    }

    s.onmessage = evt => {
      const message = JSON.parse(evt.data)
      if(message.pong || message.presence) {
	      this.addUser(message)
      } else {
	      this.addMessage(message)
	      this.addUser(message)
      }
    }

    s.onclose = (e) => {
      console.log('disconnected' + e.code)
      this.setState({ connected: false })
      switch(e.code) {
	      case 1000:
		break;
	      default:
		this.reconnect();
                break;
      }
    }
    s.onerror = (e) => {
      console.log('disconnected' + e.code)
      this.setState({ connected: false })
      switch(e.code) {
	      case 'ECONNREFUSED':
		      this.reconnect();
		      break;
	      default:
		      console.log(e);
      }
    }
  }

  reconnect = () => {
    if (this.shouldreconnect) {
    console.log("Will retry in " + i + "ms");
    var that = this;
    setTimeout(() => { 
	    console.log("Reconnecting...");
	    var nws = new WebSocket(URL);
	    that.websocketsetup(nws)
	    // that.setState({ ws: nws });
	    that.ws = nws;
    }, i);
    }
  }

  componentDidMount() {
	  var r = String(Math.floor(Math.random() * 10000));
	  this.setState(state => ({
		  id: r,
		  name: "User-" + r
	  }))
	  console.log("Component Mounting...")
  	  this.ws = new WebSocket(URL)
	  this.websocketsetup(this.ws);
    	  var that = this;
	  this.userinterval = setInterval(() => {
		  var newstate = that.state;
		  Object.keys(newstate.users).map((u, index) => {
	  		if ( (Date.now() - newstate.users[u]['time']) <= green ) {
				newstate.users[u]['color'] = "green";
	  		} else if ( (Date.now() - newstate.users[u]['time']) <= yellow ) {
				newstate.users[u]['color'] = "yellow";
			} else if ( (Date.now() - newstate.users[u]['time']) <= red ) {
				newstate.users[u]['color'] = "red";
			} else {
				delete newstate.users[u];
			}
			return true;
		   })
		   that.setState(newstate);
		   return true;
		}, 1000);
	  this.shouldreconnect = true
  }

  componentWillUnmount() {
	console.log("unloading component and disconnecting")
	this.shouldreconnect = false;
	if ( this.userinterval ) {
		clearTimeout(this.userinterval);
	}
	this.ws.close()
  }

  addMessage = message => {
    if(this.state.messages.length > 20)  {
	    this.state.messages.shift()
    }
    this.setState(state => ({ messages: [...this.state.messages, message] }))
  }

  addUser = message => {
    var newusers = this.state.users
    newusers[message.id] = { name: message.name, time: Date.now(), color: 'green' }
    this.setState(state => ({
	    users: newusers
    }))
  }

  // submitMessage = messageString => this.ws.send(JSON.stringify({ name: this.state.name, message: messageString }))
  submitMessage = (m) => {
    this.ws.send(JSON.stringify({ name: this.state.name, id: this.state.id, message: m }))
  }

  sendPresence = () => {
    this.ws.send(JSON.stringify({ name: this.state.name, id: this.state.id, presence: true }))
  }

  userColor = (u) => {
	  if ( u['color'] === 'green' ) {
		  return (<span class="greendot"></span>);
	  } else if ( u['color'] === 'yellow' ) {
		  return (<span class="yellowdot"></span>);
	  } else if ( u['color'] === 'red' ) {
		  return (<span class="reddot"></span>);
	  } else {
		  return (<span class="greydot"></span>);
	  }
  }

  render() {
    return (
      <div id="container">
	<div id="top">
	    <b>Chat</b>
	    <span class="topright">Connected: { this.state.connected ? (<span class="greendot"></span>) : (<span class="reddot"></span>) }</span>
	</div>
	<div id="middle">
	    	<div id="users">
	    		<center><b>Users</b></center>
        		<label htmlFor="name">Name:&nbsp;
			<input
		            type="text"
	        	    id={'name'}
	        	    placeholder={'Enter your name...'}
		            value={this.state.name}
		            onChange={e => this.setState({ name: e.target.value })}
		        />
		        </label>
	    	<ul>
	        {
			Object.keys(this.state.users).sort().map( (user, index) => ( 
				<li>
				{ this.state.users[user] ? this.state.users[user]['name'] : user } {this.userColor(this.state.users[user])}
				</li>
			))
		}
	    	</ul>
	    	</div>

		<div id="chatmessages">
	    	<center><b>Messages</b></center>
	        {this.state.messages.map((message, index) =>
	          <ChatMessage
	            key={index}
	            message={message.message}
	            name={ this.state.users[message.id] ? this.state.users[message.id]['name'] : message.name }
		    online={ this.state.users[message.id] ? true : false }
	          />,
	        )}
		<div id="bottom">
		<ChatInput 
		  ws={this.ws}
		  onSubmitMessage={this.submitMessage}
		/>
	    	</div>
		</div>
	</div>
      </div>
    )
  }
}

export default Chat

