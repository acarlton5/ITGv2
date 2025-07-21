import React from "react";
import { HeaderLogoContainer, MainHeader } from "../styles/headerStyles";
import { ITGLogoURL } from "../assets/constants";

const Header = () => {
  return (
    <MainHeader>
      <HeaderLogoContainer>
        <img src={ITGLogoURL} alt="ITG logo"></img>
        <h1>Project ITG</h1>
      </HeaderLogoContainer>
    </MainHeader>
  );
};

export default Header;
