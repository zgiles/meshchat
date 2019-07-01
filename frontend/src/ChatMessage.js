import React from 'react'

// <p style={ online ? ("color:gray;") : "" }>

export default ({ name, message, online }) =>
	  <p>
		{ online ? (<strong>{name}</strong>) : (<strike><strong>{name}</strong></strike>) }:&nbsp;{message}
		  </p>

